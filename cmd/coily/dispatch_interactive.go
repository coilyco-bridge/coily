package main

import (
	"context"
	"fmt"
	"os"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// Defaults for the coily <-> Warp seam. These three strings are the entire
// shared knowledge between coily and the agentic-os launch config: a path
// for the prompt scratch file, a Warp launch-config name, and a channel
// picker. agentic-os reads the scratch file and runs claude; coily writes
// the scratch file and fires the URL. Neither side knows what the other
// contains beyond those strings.
const (
	defaultDispatchScratchPath = "/tmp/coily-dispatch-prompt.txt"
	defaultDispatchLaunchName  = "claude-dispatch-interactive"
	defaultDispatchChannel     = "preview"
)

// channelScheme maps the --channel flag to the URL scheme that lands in
// that Warp build. `warp://` always opens Stable, `warppreview://` always
// opens Preview - there is no LaunchServices toggle that flips this. So
// the operator picks the channel by scheme at call site.
//
// Preview is the Mac daily driver per coilysiren/agentic-os#107 and is
// the source of the warpdotdev/Warp#9379 tab_config URI handler that
// motivated promoting Preview in the first place. Stable is the named
// fallback for when Preview wedges.
func channelScheme(channel string) (string, error) {
	switch channel {
	case "preview":
		return "warppreview", nil
	case "stable":
		return "warp", nil
	default:
		return "", fmt.Errorf("invalid --channel %q (valid values: preview | stable)", channel)
	}
}

// launchURL builds the full warp(preview)://launch/<name> URL.
func launchURL(channel, launchName string) (string, error) {
	scheme, err := channelScheme(channel)
	if err != nil {
		return "", err
	}
	return scheme + "://launch/" + launchName, nil
}

// openWarpLaunch is the seam tests swap to avoid actually spawning Warp.
// Default fires `open <url>` through the shell runner so the urfave/cli
// plumbing and audit row remain the production path.
var openWarpLaunch = func(ctx context.Context, r *Runner, url string) error {
	return r.Runner.Exec(ctx, "open", url)
}

// dispatchInteractiveCommand spins up a new Warp tab cwd'd into the target
// repo with `claude "Work on issue <ref>"` already submitted as the first
// turn. Hands control back to the operator immediately.
//
// Coily <-> Warp interface (#270). coily writes the minimal prompt
// "Work on issue <ref>" to /tmp/coily-dispatch-prompt.txt with mode 0600,
// then fires `open warppreview://launch/claude-dispatch-interactive` (or
// `warp://...` if --channel=stable). The agentic-os launch config of the
// same name parses the ref out of the scratch file to derive the local
// repo path, cd's in, and execs claude with the prompt body. No
// bidirectional coupling: the scratch path and the launch name are the
// only shared knowledge.
//
// Soft-fail: if `open` is unavailable or Warp does not respond, coily
// prints a copy-paste fallback (the exact claude invocation the operator
// can paste in a manually-opened tab) and exits 0. The scratch file is
// left in place so the launch config can still consume it on retry.
func (r *Runner) dispatchInteractiveCommand() *cli.Command {
	return &cli.Command{
		Name:      "interactive",
		Usage:     "Open a new Warp tab with `claude \"Work on issue <ref>\"` pre-submitted.",
		ArgsUsage: "<owner/repo#N | issue-url>",
		Description: `interactive validates the issue ref exists and is open, writes the prompt
"Work on issue <ref>" to a scratch file at /tmp/coily-dispatch-prompt.txt
(mode 0600), then fires open warppreview://launch/claude-dispatch-interactive
to trigger the agentic-os Warp launch config that consumes the scratch file
and execs claude inside the local checkout at ~/projects/coilysiren/<repo>.

Defaults to Warp Preview (--channel preview). Pass --channel stable to
target Warp Stable instead. Preview is the Mac daily driver; stable is
the explicit fallback for when Preview wedges.

Hands control back immediately. The dispatched session runs in the
foreground in its own Warp tab; no headless retry loop, no audit polling.
If you need an AFK fire-and-forget run, use 'coily dispatch headless'.

Soft-fails to a copy-paste fallback if Warp / open are unavailable.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "print the resolved issue + prompt + scratch path + URL, do not write the scratch file or fire open",
			},
			&cli.StringFlag{
				Name:  "scratch-path",
				Usage: "override the prompt scratch-file path. Must match the path the Warp launch config reads.",
				Value: defaultDispatchScratchPath,
			},
			&cli.StringFlag{
				Name:  "launch-name",
				Usage: "override the Warp launch-config name fired via warp(preview)://launch/<name>.",
				Value: defaultDispatchLaunchName,
			},
			&cli.StringFlag{
				Name:  "channel",
				Usage: "Warp channel to fire the URL into. `preview` (default, daily driver) or `stable` (fallback).",
				Value: defaultDispatchChannel,
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:       "dispatch.interactive",
				SkipPolicy: false,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--scratch-path": c.String("scratch-path"),
						"--launch-name":  c.String("launch-name"),
						"--channel":      c.String("channel"),
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
					return runDispatchInteractive(ctx, r, c)
				},
			},
			r.Audit,
		),
	}
}

// interactivePrompt is the minimal prompt the launch config consumes.
// Discipline matches recall-dispatch: no preamble, no URL, no flair.
// AGENTS.md in the target repo's cwd does the rest of the framing. The
// launch-config script greps "coilysiren/<repo>#<N>" out of this string
// to derive the cwd.
func interactivePrompt(ref *issueRef) string {
	return fmt.Sprintf("Work on issue %s", ref)
}

func runDispatchInteractive(ctx context.Context, r *Runner, c *cli.Command) error {
	args := c.Args().Slice()
	if len(args) != 1 {
		return fmt.Errorf("dispatch interactive: pass exactly one issue reference (got %d args)", len(args))
	}
	ref, issue, err := resolveDispatchIssue(ctx, r, args[0])
	if err != nil {
		return err
	}

	prompt := interactivePrompt(ref)
	scratchPath := c.String("scratch-path")
	launchName := c.String("launch-name")
	channel := c.String("channel")
	repoPath, _ := localRepoPath(ref.Repo)

	url, err := launchURL(channel, launchName)
	if err != nil {
		return fmt.Errorf("dispatch interactive: %w", err)
	}

	if c.Bool("dry-run") {
		fmt.Printf("# dispatch interactive (dry-run)\n")
		fmt.Printf("issue:        %s\n", ref)
		fmt.Printf("url:          %s\n", issue.URL)
		fmt.Printf("cwd:          %s\n", repoPath)
		fmt.Printf("scratch-path: %s\n", scratchPath)
		fmt.Printf("channel:      %s\n", channel)
		fmt.Printf("launch-url:   %s\n", url)
		fmt.Printf("----- prompt -----\n%s\n----- end -----\n", prompt)
		return nil
	}

	if err := writeDispatchScratchFile(scratchPath, prompt); err != nil {
		return fmt.Errorf("dispatch interactive: write scratch file %s: %w", scratchPath, err)
	}

	if err := openWarpLaunch(ctx, r, url); err != nil {
		printInteractiveFallback(ref, repoPath, prompt, url, err)
		return nil
	}
	return nil
}

// writeDispatchScratchFile writes the prompt to path with mode 0600 so
// only the running user can read it. The launch-config script deletes the
// file after consuming it; on soft-fail we leave it in place for retry.
func writeDispatchScratchFile(path, prompt string) error {
	return os.WriteFile(path, []byte(prompt+"\n"), 0o600)
}

// printInteractiveFallback emits the exact command the operator can paste
// in a manually-opened tab when the Warp URL fire fails. Soft-fail by
// design: missing Warp shouldn't error the audit row, just inform the
// operator that the automated path didn't fire.
func printInteractiveFallback(ref *issueRef, repoPath, prompt, url string, openErr error) {
	fmt.Fprintf(os.Stderr, "dispatch interactive: %s did not fire (%v).\n", url, openErr)
	fmt.Fprintf(os.Stderr, "Paste this in a new tab cwd'd to %s:\n\n", repoPath)
	fmt.Fprintf(os.Stderr, "  cd %s && claude %q\n\n", repoPath, prompt)
	fmt.Fprintf(os.Stderr, "Issue: %s\n", ref)
}
