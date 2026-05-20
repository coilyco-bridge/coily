package main

import (
	"context"
	"fmt"
	"os"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// Defaults for the coily <-> Warp seam. These two strings are the entire
// shared knowledge between coily and the agentic-os launch config: a path
// for the prompt scratch file and a Warp launch-config name. agentic-os
// reads the scratch file and runs claude; coily writes the scratch file
// and fires the URL. Neither side knows what the other contains beyond
// those two strings.
const (
	defaultDispatchScratchPath = "/tmp/coily-dispatch-prompt.txt"
	defaultDispatchLaunchName  = "claude-dispatch-interactive"
)

// openWarpLaunch is the seam tests swap to avoid actually spawning Warp.
// Default fires `open warp://launch/<name>` through the shell runner so
// the urfave/cli plumbing and audit row remain the production path.
var openWarpLaunch = func(ctx context.Context, r *Runner, launchName string) error {
	return r.Runner.Exec(ctx, "open", "warp://launch/"+launchName)
}

// dispatchInteractiveCommand spins up a new Warp tab cwd'd into the target
// repo with `claude "Work on issue <ref>"` already submitted as the first
// turn. Hands control back to the operator immediately.
//
// Coily <-> Warp interface (#270). coily writes the minimal prompt
// "Work on issue <ref>" to /tmp/coily-dispatch-prompt.txt with mode 0600,
// then fires `open warp://launch/claude-dispatch-interactive`. The
// agentic-os launch config of the same name parses the ref out of the
// scratch file to derive the local repo path, cd's in, and execs claude
// with the prompt body. No bidirectional coupling: the scratch path and
// the launch name are the only shared knowledge.
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
(mode 0600), then fires open warp://launch/claude-dispatch-interactive to
trigger the agentic-os Warp launch config that consumes the scratch file
and execs claude inside the local checkout at ~/projects/coilysiren/<repo>.

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
				Usage: "override the Warp launch-config name fired via warp://launch/<name>.",
				Value: defaultDispatchLaunchName,
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
	repoPath, _ := localRepoPath(ref.Repo)

	if c.Bool("dry-run") {
		fmt.Printf("# dispatch interactive (dry-run)\n")
		fmt.Printf("issue:        %s\n", ref)
		fmt.Printf("url:          %s\n", issue.URL)
		fmt.Printf("cwd:          %s\n", repoPath)
		fmt.Printf("scratch-path: %s\n", scratchPath)
		fmt.Printf("launch-url:   warp://launch/%s\n", launchName)
		fmt.Printf("----- prompt -----\n%s\n----- end -----\n", prompt)
		return nil
	}

	if err := writeDispatchScratchFile(scratchPath, prompt); err != nil {
		return fmt.Errorf("dispatch interactive: write scratch file %s: %w", scratchPath, err)
	}

	if err := openWarpLaunch(ctx, r, launchName); err != nil {
		printInteractiveFallback(ref, repoPath, prompt, launchName, err)
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
func printInteractiveFallback(ref *issueRef, repoPath, prompt, launchName string, openErr error) {
	fmt.Fprintf(os.Stderr, "dispatch interactive: warp://launch/%s did not fire (%v).\n", launchName, openErr)
	fmt.Fprintf(os.Stderr, "Paste this in a new tab cwd'd to %s:\n\n", repoPath)
	fmt.Fprintf(os.Stderr, "  cd %s && claude %q\n\n", repoPath, prompt)
	fmt.Fprintf(os.Stderr, "Issue: %s\n", ref)
}
