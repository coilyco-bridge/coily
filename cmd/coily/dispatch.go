package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// dispatchCommand fires a fresh top-level `claude -p` session bound to a
// real open issue in coilysiren/*. The privileged-op claim that makes this
// defensible: the prompt is not free text. It is derived from a GitHub
// issue that exists in the org. An agent (or a person) cannot launder an
// arbitrary prompt through it without first filing an issue, which leaves
// its own public trail.
//
// See coilysiren/coily#136 for the design discussion.
func (r *Runner) dispatchCommand() *cli.Command {
	return &cli.Command{
		Name:      "dispatch",
		Usage:     "Fire `claude -p` against a real open coilysiren/* issue.",
		ArgsUsage: "<owner/repo#N | issue-url>",
		Description: `dispatch resolves a GitHub issue reference, refuses anything outside the
coilysiren/* org or any issue that is not open, seeds a prompt from the
issue title and body plus the standard git-workflow footer, then exec's
claude -p inside the matching local checkout at
~/projects/coilysiren/<repo>.

Refuses to run if the local checkout is missing - the operator clones
the repo first, then re-runs dispatch.

This is the only sanctioned path from inside a coily session to a fresh
top-level claude session. The auto-mode classifier denies bare 'claude -p'
because the prompt is unconstrained; here the prompt is derived from a
real open org issue.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "print the resolved issue + seeded prompt + repo path, do not exec claude",
			},
			&cli.StringFlag{
				Name:  "claude-bin",
				Usage: "override the claude binary path (default: 'claude' on $PATH)",
				Value: "claude",
			},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name:       "dispatch",
				SkipPolicy: false,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--claude-bin": c.String("claude-bin"),
					}, c.Args().Slice()
				},
				CommitScopeArgvHint: func(argv []string) string {
					ref := firstIssueRef(argv)
					if ref == nil {
						return ""
					}
					p, err := localRepoPath(ref.Repo)
					if err != nil {
						return ""
					}
					return p
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return runDispatch(ctx, r, c)
				},
			},
			r.Audit,
		),
	}
}

// issueRef is the parsed shape of a GitHub issue reference.
type issueRef struct {
	Owner  string
	Repo   string
	Number int
}

func (i issueRef) String() string {
	return fmt.Sprintf("%s/%s#%d", i.Owner, i.Repo, i.Number)
}

// issueRefShortRE matches owner/repo#N.
var issueRefShortRE = regexp.MustCompile(`^([A-Za-z0-9._-]+)/([A-Za-z0-9._-]+)#(\d+)$`)

// issueRefURLRE matches https://github.com/owner/repo/issues/N.
var issueRefURLRE = regexp.MustCompile(`^https?://github\.com/([A-Za-z0-9._-]+)/([A-Za-z0-9._-]+)/issues/(\d+)/?$`)

// parseIssueRef accepts the two reference forms and returns the parts.
func parseIssueRef(s string) (*issueRef, error) {
	s = strings.TrimSpace(s)
	for _, re := range []*regexp.Regexp{issueRefShortRE, issueRefURLRE} {
		if m := re.FindStringSubmatch(s); m != nil {
			n := 0
			if _, err := fmt.Sscanf(m[3], "%d", &n); err != nil {
				return nil, fmt.Errorf("dispatch: parse issue number in %q: %w", s, err)
			}
			if n <= 0 {
				return nil, fmt.Errorf("dispatch: issue number must be positive: %q", s)
			}
			return &issueRef{Owner: m[1], Repo: m[2], Number: n}, nil
		}
	}
	return nil, fmt.Errorf("dispatch: not an issue reference (want owner/repo#N or https://github.com/owner/repo/issues/N): %q", s)
}

// firstIssueRef scans argv for the first arg that parses as an issue ref.
// Used by the CommitScopeArgvHint to bind the audit row to the target repo
// before the action runs.
func firstIssueRef(argv []string) *issueRef {
	for _, a := range argv {
		if ref, err := parseIssueRef(a); err == nil {
			return ref
		}
	}
	return nil
}

// allowedOwner is the org coily will dispatch against. Hard-coded rather
// than configurable: this is the security claim, not a knob.
const allowedOwner = "coilysiren"

// localRepoPath returns the expected local checkout for a coilysiren repo.
// Mirrors the workspace shape from AGENTS.md.
func localRepoPath(repo string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "projects", allowedOwner, repo), nil
}

// ghIssue is the subset of the GitHub REST issues response dispatch
// consumes. Fetched via `gh api /repos/{owner}/{repo}/issues/{N}` rather
// than `gh issue view --json`, since the latter routes through GraphQL and
// shares its tight secondary-rate-limit budget. REST returns state as
// lowercase ("open"/"closed"); the State-check downstream is case-insensitive.
type ghIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	URL    string `json:"html_url"`
}

func runDispatch(ctx context.Context, r *Runner, c *cli.Command) error {
	args := c.Args().Slice()
	if len(args) != 1 {
		return fmt.Errorf("dispatch: pass exactly one issue reference (got %d args)", len(args))
	}
	ref, err := parseIssueRef(args[0])
	if err != nil {
		return err
	}
	if ref.Owner != allowedOwner {
		return fmt.Errorf("dispatch: refusing to dispatch outside %s/* (got %s)", allowedOwner, ref.Owner)
	}

	repoPath, err := localRepoPath(ref.Repo)
	if err != nil {
		return fmt.Errorf("dispatch: resolve local repo path: %w", err)
	}
	if st, statErr := os.Stat(repoPath); statErr != nil || !st.IsDir() {
		return fmt.Errorf("dispatch: local checkout missing at %s. Clone it first: gh repo clone %s/%s %s",
			repoPath, ref.Owner, ref.Repo, repoPath)
	}

	issue, err := fetchIssue(ctx, r, ref)
	if err != nil {
		return err
	}
	if !strings.EqualFold(issue.State, "OPEN") {
		return fmt.Errorf("dispatch: refusing to dispatch against non-open issue %s (state=%s)", ref, issue.State)
	}

	prompt := seedPrompt(ref, issue)

	if c.Bool("dry-run") {
		fmt.Printf("# dispatch (dry-run)\n")
		fmt.Printf("issue: %s\n", ref)
		fmt.Printf("url:   %s\n", issue.URL)
		fmt.Printf("cwd:   %s\n", repoPath)
		fmt.Printf("----- seeded prompt -----\n%s\n----- end -----\n", prompt)
		return nil
	}

	bin := c.String("claude-bin")
	if bin == "" {
		bin = "claude"
	}
	return r.Runner.ExecIn(ctx, repoPath, bin, "-p", prompt)
}

// fetchIssue shells out to gh to resolve the issue. Uses the runner's
// Capture so stderr is forwarded if gh fails. Goes through the REST API
// (`gh api /repos/.../issues/N`) rather than `gh issue view --json` so it
// doesn't share GraphQL's tight secondary-rate-limit budget (coilysiren/coily#138).
func fetchIssue(ctx context.Context, r *Runner, ref *issueRef) (*ghIssue, error) {
	raw, err := r.Runner.Capture(ctx, "gh", "api",
		fmt.Sprintf("/repos/%s/%s/issues/%d", ref.Owner, ref.Repo, ref.Number),
	)
	if err != nil {
		return nil, fmt.Errorf("dispatch: gh api repos/%s/%s/issues/%d: %w", ref.Owner, ref.Repo, ref.Number, err)
	}
	var issue ghIssue
	if err := json.Unmarshal(raw, &issue); err != nil {
		return nil, fmt.Errorf("dispatch: parse gh api issue output: %w", err)
	}
	return &issue, nil
}

// seedPrompt composes the prompt fed to claude -p. Footer carries the
// invariants from AGENTS.md (commit to main, close with closes #N, push,
// never --no-verify). Kept as a single string constant so the wording
// drifts in one place rather than per-call.
func seedPrompt(ref *issueRef, issue *ghIssue) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Work on GitHub issue %s.\n\n", ref)
	fmt.Fprintf(&b, "Title: %s\n", issue.Title)
	fmt.Fprintf(&b, "URL:   %s\n\n", issue.URL)
	fmt.Fprintf(&b, "Issue body:\n\n%s\n\n", strings.TrimSpace(issue.Body))
	fmt.Fprintf(&b, "%s", dispatchFooter)
	return b.String()
}

const dispatchFooter = `Workflow rules (from AGENTS.md):
- Commit to main directly. Push after each commit. No PRs unless asked.
- Run tests, linters, and builds without asking. Fix failures.
- Never use --no-verify.
- Close the issue with a commit trailer: closes #` + `<N>` + ` (or fixes / resolves).
- Cwd is the target repo. Stay inside it.
`
