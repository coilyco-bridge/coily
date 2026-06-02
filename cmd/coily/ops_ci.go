package main

// ops_ci.go is the user-facing CI run-status watcher built on the
// github.com HTML reader from gh_actions_web.go (coilysiren/coily#291).
//
// `coily ops ci <owner/repo>` reads recent Actions runs off the
// github.com web page, which draws on a rate-limit pool separate from
// the api.github.com REST 5000/hr budget. So this keeps working when a
// burst of REST traffic has exhausted the personal-PAT budget - the
// motivating problem behind #289.
//
// This is a standalone watcher, not the transparent gh-read-path
// rewrite. The full tiered fallback (App token -> web -> personal PAT)
// is slice C (#293); this command is the visible, runnable first cut.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// ciWatchMinInterval floors the --watch poll interval. Even off the
// separate web budget, hammering github.com faster than this is rude
// and pointless - CI status does not move that fast.
const ciWatchMinInterval = 3 * time.Second

func (r *Runner) ciCommand() *cli.Command {
	return &cli.Command{
		Name:      "ci",
		Usage:     "Watch GitHub Actions run status, read off the github.com web budget.",
		ArgsUsage: "<owner/repo>",
		Description: `ci reads recent GitHub Actions runs for a repo by parsing the
server-rendered github.com Actions page, instead of the api.github.com
REST API. The web page draws on a rate-limit pool separate from the
REST 5000/hr budget, so this keeps reporting CI status even when a
burst of REST traffic has exhausted the personal-token budget.

  coily ops ci coilysiren/coily            one-shot table of recent runs
  coily ops ci coilysiren/coily --watch    re-poll until interrupted
  coily ops ci coilysiren/coily --json     raw REST-shaped JSON

Public repos need no auth. For a private repo, export a github.com
session cookie as COILY_GH_WEB_COOKIE; without it the read fails and
you should fall back to coily ops gh.

This parses server-rendered HTML and is best-effort: it will need a
touch-up when GitHub reshapes the Actions page. That tradeoff is the
point - it buys a budget the REST API cannot.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "watch",
				Aliases: []string{"w"},
				Usage:   "re-poll continuously until interrupted",
			},
			&cli.DurationFlag{
				Name:  "interval",
				Value: 10 * time.Second,
				Usage: "poll interval for --watch",
			},
			&cli.IntFlag{
				Name:  "limit",
				Value: 20,
				Usage: "maximum runs to show",
			},
			&cli.StringFlag{
				Name:  "branch",
				Usage: "show only runs on this branch",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "emit raw REST-shaped JSON instead of the table",
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.ci",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--branch": c.String("branch")}, c.Args().Slice()
				},
				Action: runCI,
			},
			r.Audit,
		),
	}
}

func runCI(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() != 1 {
		return fmt.Errorf("ops ci: want exactly one <owner/repo> argument, got %d", c.Args().Len())
	}
	owner, repo, ok := splitOwnerRepo(c.Args().First())
	if !ok {
		return fmt.Errorf("ops ci: %q is not an owner/repo slug", c.Args().First())
	}
	cookie := os.Getenv(ghWebCookieEnv)
	opts := ciRenderOpts{
		limit:    c.Int("limit"),
		branch:   c.String("branch"),
		jsonMode: c.Bool("json"),
	}

	if !c.Bool("watch") {
		return ciPollOnce(ctx, owner, repo, cookie, opts)
	}

	interval := c.Duration("interval")
	if interval < ciWatchMinInterval {
		interval = ciWatchMinInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		// Clear screen so the watch view refreshes in place rather than
		// scrolling. The header carries the timestamp so a frozen poll is
		// obvious.
		fmt.Print("\033[H\033[2J")
		fmt.Printf("coily ops ci %s/%s  -  %s  (every %s, Ctrl-C to stop)\n\n",
			owner, repo, time.Now().Format("15:04:05"), interval)
		if err := ciPollOnce(ctx, owner, repo, cookie, opts); err != nil {
			fmt.Fprintf(os.Stderr, "poll failed: %v\n", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// ciRenderOpts carries the display knobs through a single poll.
type ciRenderOpts struct {
	limit    int
	branch   string
	jsonMode bool
}

func ciPollOnce(ctx context.Context, owner, repo, cookie string, opts ciRenderOpts) error {
	raw, err := readActionsRunsViaWeb(ctx, owner, repo, cookie)
	if err != nil {
		return err
	}
	if opts.jsonMode {
		fmt.Println(string(raw))
		return nil
	}
	var result webRunsResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return fmt.Errorf("ops ci: decode web result: %w", err)
	}
	renderCITable(os.Stdout, result.WorkflowRuns, opts)
	return nil
}

// renderCITable prints an aligned table of runs, newest first as the
// page already orders them, after the branch filter and limit.
func renderCITable(out *os.File, runs []webWorkflowRun, opts ciRenderOpts) {
	shown := 0
	tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "  STATUS\tWORKFLOW\tRUN\tBRANCH\tAGE\tTITLE")
	for _, run := range runs {
		if opts.branch != "" && run.HeadBranch != opts.branch {
			continue
		}
		if opts.limit > 0 && shown >= opts.limit {
			break
		}
		shown++
		_, _ = fmt.Fprintf(tw, "%s\t%s\t#%d\t%s\t%s\t%s\n",
			ciStatusGlyph(run.Status, run.Conclusion),
			truncate(run.Name, 22),
			run.RunNumber,
			truncate(run.HeadBranch, 24),
			ciAge(run.CreatedAt),
			truncate(run.DisplayTitle, 50),
		)
	}
	_ = tw.Flush()
	if shown == 0 {
		_, _ = fmt.Fprintln(out, "(no runs)")
	}
}

// ciStatusGlyph maps a (status, conclusion) pair to a compact symbol
// plus word. Completed runs key off the conclusion; everything else
// keys off the status.
func ciStatusGlyph(status, conclusion string) string {
	if status == "completed" {
		switch conclusion {
		case "success":
			return "✓ pass"
		case "failure":
			return "✗ fail"
		case "cancelled":
			return "⊘ cancel"
		case "skipped":
			return "⊘ skip"
		case "timed_out":
			return "✗ timeout"
		default:
			return "• " + conclusion
		}
	}
	switch status {
	case "in_progress":
		return "● run"
	case "queued":
		return "○ queued"
	default:
		return "• " + status
	}
}

// ciAge renders an ISO-8601 timestamp as a coarse relative age. An
// unparseable timestamp falls through to the raw string rather than
// erroring - the age column is informational.
func ciAge(iso string) string {
	if iso == "" {
		return "?"
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// splitOwnerRepo parses an "owner/repo" slug. Both halves must be
// non-empty and neither may itself contain a slash.
func splitOwnerRepo(slug string) (owner, repo string, ok bool) {
	owner, repo, found := strings.Cut(slug, "/")
	if !found || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return "", "", false
	}
	return owner, repo, true
}

// truncate clips s to n runes, appending an ellipsis when it had to cut.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
