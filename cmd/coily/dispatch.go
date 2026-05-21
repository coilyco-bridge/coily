package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/coilysiren/cli-guard/ghcache"
	"github.com/coilysiren/cli-guard/ghratelimit"
	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// dispatchCommand is the umbrella verb. It refuses bare invocation and
// requires an explicit mode subverb (`headless` or `interactive`). The
// security claim that makes this verb defensible is unchanged from #136:
// the prompt is derived from a real open coilysiren/* issue, not free text.
//
// Why force the mode choice. Both modes have valid uses. Headless is for
// AFK queue work and bulk fan-out (subscription bill). Interactive is for
// "this should be its own focused session but I want eyes on it" (Anthropic
// API key bill, foreground supervision). A default would silently pick a
// billing model and a supervision posture. Operator chooses on every call.
//
// See coilysiren/coily#270 for the split design.
func (r *Runner) dispatchCommand() *cli.Command {
	return &cli.Command{
		Name:      "dispatch",
		Usage:     "Fire claude against a real open coilysiren/* issue. Mode required: headless | interactive.",
		ArgsUsage: "<headless|interactive> <owner/repo#N | issue-url>",
		Description: `dispatch resolves a GitHub issue reference, refuses anything outside the
coilysiren/* org or any issue that is not open, then hands off to one of
two mode subverbs:

  coily dispatch headless    <ref>   Spawn a detached claude -p in the
                                     local checkout, log to a file, return
                                     immediately. AFK queue work.
  coily dispatch interactive <ref>   Open a new Warp tab cwd'd into the
                                     repo with claude pre-submitted with
                                     "Work on issue <ref>". Operator has
                                     eyes on it.

It also carries one maintenance verb:

  coily dispatch reap                Remove dispatch worktrees whose
                                     branch is already merged into main.

Bare 'coily dispatch <ref>' errors. Pick a mode.`,
		Commands: []*cli.Command{
			r.dispatchHeadlessCommand(),
			r.dispatchInteractiveCommand(),
			r.dispatchReapCommand(),
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			return fmt.Errorf("dispatch: specify mode: interactive | headless (see `coily dispatch --help`)")
		},
	}
}

// dispatchHeadlessCommand fires `claude -p` against the issue in the
// local checkout as a fully detached, fire-and-forget child: new session,
// own process group, stdio to a per-dispatch log file. coily resolves the
// issue, spawns the child, prints its PID and log path, and returns
// immediately (coilysiren/coily#302).
func (r *Runner) dispatchHeadlessCommand() *cli.Command {
	return &cli.Command{
		Name:      "headless",
		Usage:     "Spawn a detached `claude -p` against a real open coilysiren/* issue.",
		ArgsUsage: "<owner/repo#N | issue-url>",
		Description: `headless resolves a GitHub issue reference, refuses anything outside the
coilysiren/* org or any issue that is not open, seeds a prompt from the
issue title and body plus the standard git-workflow footer, then spawns
claude -p inside the matching local checkout at
~/projects/coilysiren/<repo>.

The child is fully detached: it runs in a new session with its own
process group, its stdio goes to a per-dispatch log file under
~/projects/coilysiren/.dispatch-logs/<repo>/ rather than the parent
terminal, and coily does not wait on it. dispatch prints the child PID
and log path and returns immediately - the run survives coily's terminal
closing. This is what makes headless truly fire-and-forget.

Refuses to run if the local checkout is missing - the operator clones
the repo first, then re-runs dispatch.

The child claude session is launched with --permission-mode (default 'auto')
and --allowedTools (default Bash,Read,Edit,Write,Glob,Grep,TodoWrite). Both
are overridable per invocation via --permission-mode and --allowed-tools so
target repos that need a wider tool set can opt in without editing dispatch.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "print the resolved issue + seeded prompt + repo path + flags, do not exec claude",
			},
			&cli.StringFlag{
				Name:  "claude-bin",
				Usage: "override the claude binary path (default: 'claude' on $PATH)",
				Value: "claude",
			},
			&cli.StringFlag{
				Name:  "permission-mode",
				Usage: "permission mode passed to the child claude session (auto, default, acceptEdits, plan, bypassPermissions). Headless dispatch needs a non-prompting mode; default is auto.",
				Value: defaultDispatchPermissionMode,
			},
			&cli.StringFlag{
				Name:  "allowed-tools",
				Usage: "comma-separated allowedTools list passed to the child. Default covers the workflow footer: git, coily wrappers, file edits, reads, todos.",
				Value: defaultDispatchAllowedTools,
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:       "dispatch.headless",
				SkipPolicy: false,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--claude-bin":      c.String("claude-bin"),
						"--permission-mode": c.String("permission-mode"),
						"--allowed-tools":   c.String("allowed-tools"),
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
					return runDispatchHeadless(ctx, r, c)
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

// defaultDispatchPermissionMode is the strictest mode that still lets the
// headless child make progress without an operator to approve prompts.
// auto matches the classifier-driven mode coily's lockdown reasons about;
// strictly stricter modes (default, plan) stall on the first non-allowed
// tool because there's no human to approve.
const defaultDispatchPermissionMode = "auto"

// defaultDispatchAllowedTools is the baseline tool set the seeded prompt
// footer assumes. It covers the workflow that closes a coilysiren/* issue:
// git read+commit+push, coily-wrapped privileged ops (gh, aws, kubectl
// pass-throughs), and the standard file/search/todos primitives. Override
// with --allowed-tools when a target repo needs more.
const defaultDispatchAllowedTools = "Bash,Read,Edit,Write,Glob,Grep,TodoWrite"

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

// resolveDispatchIssue parses the ref, refuses non-coilysiren and
// non-open issues, and returns both the ref and the fetched issue. Shared
// by headless and interactive subverbs so the security claim and validation
// shape stay identical.
func resolveDispatchIssue(ctx context.Context, r *Runner, raw string) (*issueRef, *ghIssue, error) {
	ref, err := parseIssueRef(raw)
	if err != nil {
		return nil, nil, err
	}
	if ref.Owner != allowedOwner {
		return nil, nil, fmt.Errorf("dispatch: refusing to dispatch outside %s/* (got %s)", allowedOwner, ref.Owner)
	}
	issue, err := fetchIssue(ctx, r, ref)
	if err != nil {
		return nil, nil, err
	}
	if !strings.EqualFold(issue.State, "OPEN") {
		return nil, nil, fmt.Errorf("dispatch: refusing to dispatch against non-open issue %s (state=%s)", ref, issue.State)
	}
	return ref, issue, nil
}

func runDispatchHeadless(ctx context.Context, r *Runner, c *cli.Command) error {
	args := c.Args().Slice()
	if len(args) != 1 {
		return fmt.Errorf("dispatch headless: pass exactly one issue reference (got %d args)", len(args))
	}
	ref, issue, err := resolveDispatchIssue(ctx, r, args[0])
	if err != nil {
		return err
	}

	repoPath, err := localRepoPath(ref.Repo)
	if err != nil {
		return fmt.Errorf("dispatch headless: resolve local repo path: %w", err)
	}
	if st, statErr := os.Stat(repoPath); statErr != nil || !st.IsDir() {
		return fmt.Errorf("dispatch headless: local checkout missing at %s. Clone it first: gh repo clone %s/%s %s",
			repoPath, ref.Owner, ref.Repo, repoPath)
	}

	prompt := seedPrompt(ref, issue)
	permMode := c.String("permission-mode")
	allowedTools := c.String("allowed-tools")

	if c.Bool("dry-run") {
		fmt.Printf("# dispatch headless (dry-run)\n")
		fmt.Printf("issue:           %s\n", ref)
		fmt.Printf("url:             %s\n", issue.URL)
		fmt.Printf("cwd:             %s\n", repoPath)
		fmt.Printf("permission-mode: %s\n", permMode)
		fmt.Printf("allowed-tools:   %s\n", allowedTools)
		fmt.Printf("----- seeded prompt -----\n%s\n----- end -----\n", prompt)
		return nil
	}

	// Pre-trust the checkout so the headless child never stalls on the
	// folder-trust prompt. Soft-fail (coilysiren/coily#290).
	if err := ensureClaudeFolderTrust(repoPath); err != nil {
		fmt.Fprintf(os.Stderr, "dispatch headless: could not pre-trust %s in ~/.claude.json (%v).\n", repoPath, err)
	}

	bin, argv := buildDispatchClaudeArgv(c.String("claude-bin"), prompt, permMode, allowedTools)

	logPath, err := dispatchLogPath(ref.Repo, ref.Number)
	if err != nil {
		return fmt.Errorf("dispatch headless: resolve log path: %w", err)
	}
	pid, err := spawnDetachedClaude(repoPath, logPath, bin, argv, r.Runner.Env)
	if err != nil {
		return fmt.Errorf("dispatch headless: %w", err)
	}
	fmt.Printf("dispatch headless: spawned claude for %s (pid %d)\n", ref, pid)
	fmt.Printf("  cwd: %s\n", repoPath)
	fmt.Printf("  log: %s\n", logPath)
	fmt.Printf("  detached - survives this terminal closing. Follow with: tail -f %s\n", logPath)
	return nil
}

// dispatchLogRootOverride lets tests redirect the headless dispatch log
// root to a tempdir. Empty in production. Read by dispatchLogRoot.
var dispatchLogRootOverride = ""

// dispatchLogRoot is the parent directory for headless dispatch logs.
// Lives alongside the dispatch-worktree root, outside any repo, so the
// detached child's stdio lands somewhere stable rather than the parent
// tty (coilysiren/coily#302).
func dispatchLogRoot() (string, error) {
	if dispatchLogRootOverride != "" {
		return dispatchLogRootOverride, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "projects", allowedOwner, ".dispatch-logs"), nil
}

// dispatchLogPath returns the per-dispatch log file path. Format:
// <root>/<repo>/issue-<N>-<YYYYMMDD-HHMMSS>.log. The timestamp keeps
// re-dispatches of the same issue from clobbering an earlier run's log.
func dispatchLogPath(repo string, number int) (string, error) {
	root, err := dispatchLogRoot()
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("issue-%d-%s.log", number, time.Now().Format("20060102-150405"))
	return filepath.Join(root, repo, name), nil
}

// spawnDetachedClaude starts `claude -p` fully detached from coily: a new
// session (see detachSysProcAttr), its own process group, stdio
// redirected to logPath instead of the parent tty, and no Wait(). The
// child survives the parent terminal closing - that is what makes
// headless dispatch fire-and-forget (coilysiren/coily#302). Returns the
// child PID. Seam: tests swap it to avoid spawning a real process.
var spawnDetachedClaude = func(repoPath, logPath, bin string, argv, env []string) (int, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return 0, fmt.Errorf("mkdir dispatch log dir: %w", err)
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, fmt.Errorf("open dispatch log %s: %w", logPath, err)
	}
	defer logFile.Close()

	binPath, err := exec.LookPath(bin)
	if err != nil {
		return 0, fmt.Errorf("resolve %s on PATH: %w", bin, err)
	}

	// Plain exec.Command, not CommandContext: the child must outlive the
	// coily process, so it must not be bound to coily's context.
	cmd := exec.Command(binPath, argv...)
	cmd.Dir = repoPath
	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = detachSysProcAttr()
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start detached claude: %w", err)
	}
	pid := cmd.Process.Pid
	// Release the child handle so coily does not have to Wait() and the
	// child is reparented to init rather than zombied when coily exits.
	if err := cmd.Process.Release(); err != nil {
		return pid, fmt.Errorf("release detached claude (pid %d): %w", pid, err)
	}
	return pid, nil
}

// buildDispatchClaudeArgv resolves the child claude binary and argv. Pulled
// out of runDispatchHeadless so the per-flag conditionals don't push the
// latter past the gocyclo threshold.
func buildDispatchClaudeArgv(claudeBin, prompt, permMode, allowedTools string) (string, []string) {
	if claudeBin == "" {
		claudeBin = "claude"
	}
	argv := []string{"-p", prompt}
	if permMode != "" {
		argv = append(argv, "--permission-mode", permMode)
	}
	if allowedTools != "" {
		argv = append(argv, "--allowedTools", allowedTools)
	}
	return claudeBin, argv
}

// fetchIssue shells out to gh to resolve the issue. Uses the runner's
// Capture so stderr is forwarded if gh fails. Goes through the REST API
// (`gh api /repos/.../issues/N`) rather than `gh issue view --json` so it
// doesn't share GraphQL's tight secondary-rate-limit budget (coilysiren/coily#138).
func fetchIssue(ctx context.Context, r *Runner, ref *issueRef) (*ghIssue, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d", ref.Owner, ref.Repo, ref.Number)
	raw, err := ghcache.GetJSON(path, func() ([]byte, error) {
		return ghratelimit.Retry(func() ([]byte, error) {
			return r.Runner.Capture(ctx, "gh", "api", path)
		})
	})
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
