package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// Defaults for the coily <-> Warp seam. The shim at
// agentic-os/warp/launch_configurations/claude-dispatch-interactive.sh
// reads from /tmp/coily-dispatch-queue/, the URI fires
// warp(preview)://(tab_config|launch)/claude-dispatch-interactive.
// Neither side knows what the other contains beyond the queue dir, the
// JSON schema, and the launch name (coilysiren/coily#280).
const (
	defaultDispatchQueueDir   = "/tmp/coily-dispatch-queue"
	defaultDispatchLaunchName = "claude-dispatch-interactive"
	defaultDispatchChannel    = "preview"
	defaultDispatchSurface    = "tab"
)

// dispatchQueueSchemaVersion lets the shim reject queue entries it does
// not understand once the schema evolves. Bump when adding required
// fields; the shim ignores forward-additive fields.
const dispatchQueueSchemaVersion = 1

// dispatchQueueEntry is the JSON payload coily writes for the shim to
// consume. The shim reads ref, title, cwd, prompt via jq. Field tags
// are the wire format - rename with care.
type dispatchQueueEntry struct {
	SchemaVersion int    `json:"schema_version"`
	Ref           string `json:"ref"`
	Title         string `json:"title"`
	Cwd           string `json:"cwd"`
	Prompt        string `json:"prompt"`
}

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

// surfacePath maps the --surface flag to the URI path segment Warp uses
// to pick between a new-tab fire (tab_config) and a new-window fire
// (launch). tab is the default per coilysiren/coily#274. The tab_config
// URI handler landed in warpdotdev/Warp#9379 and was first available in
// Preview builds dated 2026-05-13 or later, Stable builds dated
// 2026-05-15 or later; --surface window stays as the explicit fallback.
func surfacePath(surface string) (string, error) {
	switch surface {
	case "tab":
		return "tab_config", nil
	case "window":
		return "launch", nil
	default:
		return "", fmt.Errorf("invalid --surface %q (valid values: tab | window)", surface)
	}
}

// dispatchURL builds the full warp(preview)://(tab_config|launch)/<name>
// URL fired by `open`. Channel picks the scheme, surface picks the path.
func dispatchURL(channel, surface, launchName string) (string, error) {
	scheme, err := channelScheme(channel)
	if err != nil {
		return "", err
	}
	path, err := surfacePath(surface)
	if err != nil {
		return "", err
	}
	return scheme + "://" + path + "/" + launchName, nil
}

// openWarpLaunch is the seam tests swap to avoid actually spawning Warp.
// Default fires `open <url>` through the shell runner so the urfave/cli
// plumbing and audit row remain the production path.
var openWarpLaunch = func(ctx context.Context, r *Runner, url string) error {
	return r.Runner.Exec(ctx, "open", url)
}

// runWorktreeAdd is the seam tests swap to avoid actually shelling out to
// git. Default runs `git -C <repoPath> worktree add -B <branch>
// <worktreePath>` through the shell runner.
var runWorktreeAdd = func(ctx context.Context, r *Runner, repoPath, branch, worktreePath string) error {
	_, err := r.Runner.Capture(ctx, "git", "-C", repoPath, "worktree", "add", "-B", branch, worktreePath)
	return err
}

// dispatchWorktreeRootOverride lets tests redirect the worktree root to
// a tempdir so they don't have to mutate ~/projects/coilysiren. Empty in
// production. Read by dispatchWorktreeRoot.
var dispatchWorktreeRootOverride = ""

// dispatchWorktreeRoot is the parent directory under which each dispatch
// gets its own worktree. Lives outside any repo so no per-repo .gitignore
// churn. One subdirectory per repo, one worktree per issue.
func dispatchWorktreeRoot() (string, error) {
	if dispatchWorktreeRootOverride != "" {
		return dispatchWorktreeRootOverride, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "projects", allowedOwner, ".dispatch-worktrees"), nil
}

// dispatchWorktreePath returns the worktree path for a given repo + issue
// number. Format: ~/projects/coilysiren/.dispatch-worktrees/<repo>/issue-<N>.
func dispatchWorktreePath(repo string, number int) (string, error) {
	root, err := dispatchWorktreeRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, repo, fmt.Sprintf("issue-%d", number)), nil
}

// dispatchWorktreeBranch returns the branch name for a given issue number.
// Format: dispatch/issue-<N>. Predictable so re-dispatching the same issue
// reuses the same branch (idempotency).
func dispatchWorktreeBranch(number int) string {
	return fmt.Sprintf("dispatch/issue-%d", number)
}

// ensureDispatchWorktree returns worktreePath after guaranteeing a git
// worktree exists there. Idempotent: if a `.git` entry already lives at
// worktreePath, returns the path without touching git. Otherwise creates
// the parent dir and runs `git worktree add -B <branch>`. The `-B` flag
// makes the branch creation idempotent too: a re-dispatch after the
// worktree was manually removed will reset the branch rather than fail.
func ensureDispatchWorktree(ctx context.Context, r *Runner, repoPath string, ref *issueRef) (string, error) {
	worktreePath, err := dispatchWorktreePath(ref.Repo, ref.Number)
	if err != nil {
		return "", fmt.Errorf("resolve worktree path: %w", err)
	}
	if _, err := os.Stat(filepath.Join(worktreePath, ".git")); err == nil {
		return worktreePath, nil
	}
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return "", fmt.Errorf("mkdir worktree parent: %w", err)
	}
	branch := dispatchWorktreeBranch(ref.Number)
	if err := runWorktreeAdd(ctx, r, repoPath, branch, worktreePath); err != nil {
		return "", fmt.Errorf("git worktree add -B %s %s (in %s): %w", branch, worktreePath, repoPath, err)
	}
	return worktreePath, nil
}

// dispatchInteractiveCommand spins up a new Warp tab cwd'd into the target
// repo with `claude "Work on issue <ref>"` already submitted as the first
// turn. Hands control back to the operator immediately.
//
// Coily <-> Warp interface (coilysiren/coily#280). coily writes one JSON
// file per dispatch under /tmp/coily-dispatch-queue/ named
// <unix-nanos>-<8hex>.json, mode 0600, carrying ref / title / cwd /
// prompt. Then fires `open warppreview://tab_config/claude-dispatch-interactive`
// (or one of the four URL shapes from channel x surface). The shim at
// agentic-os/warp/launch_configurations/claude-dispatch-interactive.sh
// acquires a queue mutex, pops the oldest entry, echoes the title header,
// and execs claude. The FIFO queue is what lets concurrent dispatches
// land in separate tabs without racing on a single scratch path.
//
// Soft-fail: if `open` is unavailable or Warp does not respond, coily
// prints a copy-paste fallback (the exact claude invocation the operator
// can paste in a manually-opened tab) and exits 0. The queue entry is
// left in place so the shim can still consume it on retry.
func (r *Runner) dispatchInteractiveCommand() *cli.Command {
	return &cli.Command{
		Name:      "interactive",
		Usage:     "Open a new Warp tab with `claude \"Work on issue <ref>\"` pre-submitted.",
		ArgsUsage: "<owner/repo#N | issue-url>",
		Description: `interactive validates the issue ref exists and is open, writes a
JSON queue entry under /tmp/coily-dispatch-queue/ carrying ref, title,
cwd, and prompt (mode 0600), then fires
open warppreview://tab_config/claude-dispatch-interactive to trigger the
agentic-os shim that pops the queue, echoes the title header, and
execs claude inside the local checkout at ~/projects/coilysiren/<repo>.

Concurrent dispatches land in separate Warp tabs without racing on a
single scratch path - the FIFO queue gives each shim its own entry
(coilysiren/coily#280).

Defaults to Warp Preview (--channel preview). Pass --channel stable to
target Warp Stable instead. Preview is the Mac daily driver, stable is
the explicit fallback for when Preview wedges.

Defaults to a new tab inside the active window (--surface tab) via the
tab_config URI handler from warpdotdev/Warp#9379. Pass --surface window
to fall back to the legacy launch_configurations path which opens a
fresh Warp window instead.

Hands control back immediately. The dispatched session runs in the
foreground in its own Warp tab or window depending on --surface, with no
headless retry loop and no audit polling. If you need an AFK
fire-and-forget run, use 'coily dispatch headless'.

Soft-fails to a copy-paste fallback if Warp / open are unavailable.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "print the resolved issue + queue entry + URL, do not write the queue entry or fire open",
			},
			&cli.StringFlag{
				Name:  "queue-dir",
				Usage: "override the dispatch queue directory. Must match the path the Warp shim reads.",
				Value: defaultDispatchQueueDir,
			},
			&cli.StringFlag{
				Name:  "launch-name",
				Usage: "override the Warp config name fired via warp(preview)://(tab_config|launch)/<name>. The same name resolves under both surfaces because the .toml and .yaml are siblings.",
				Value: defaultDispatchLaunchName,
			},
			&cli.StringFlag{
				Name:  "channel",
				Usage: "Warp channel to fire the URL into. `preview` (default, daily driver) or `stable` (fallback).",
				Value: defaultDispatchChannel,
			},
			&cli.StringFlag{
				Name:  "surface",
				Usage: "URI path that picks tab vs window fire. `tab` (default, opens a new tab via warpdotdev/Warp#9379) or `window` (fallback, opens a fresh window via the legacy launch_configurations path).",
				Value: defaultDispatchSurface,
			},
			&cli.BoolFlag{
				Name:  "no-worktree",
				Usage: "skip git worktree creation and cd the dispatched session straight into the bare repo checkout. Escape hatch for sessions that must work against the canonical tree; concurrent dispatches into the same repo will race on the working tree.",
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:       "dispatch.interactive",
				SkipPolicy: false,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--queue-dir":   c.String("queue-dir"),
						"--launch-name": c.String("launch-name"),
						"--channel":     c.String("channel"),
						"--surface":     c.String("surface"),
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

// interactivePrompt is the prompt the shim execs claude with. Embeds
// the first-action instruction so the dispatched agent fetches issue
// body + comments before touching code; without the prime the agent
// often skips the explicit fetch and works from the bare ref line
// (coilysiren/coily#279).
func interactivePrompt(ref *issueRef, issue *ghIssue) string {
	return fmt.Sprintf(
		"Work on issue %s. First action: run `coily ops gh issue view %s --comments` and read the full body and comment thread before doing anything else.",
		ref, issue.URL,
	)
}

// interactiveTitleLine is the self-identifying header the shim echoes
// in the dispatched tab before exec'ing claude. Gives the operator an
// at-a-glance read on what the session is working on without waiting
// for claude's first turn (coilysiren/coily#279).
func interactiveTitleLine(ref *issueRef, issue *ghIssue) string {
	return fmt.Sprintf("%s: %s", ref, strings.TrimSpace(issue.Title))
}

// resolveDispatchCwd returns the directory the dispatched session should
// run in. With --no-worktree, that's the bare repo checkout. Otherwise
// it's the per-issue worktree path: dry-run resolves the path without
// touching git, normal mode creates the worktree (or reuses an existing
// one).
func resolveDispatchCwd(ctx context.Context, r *Runner, repoPath string, ref *issueRef, noWorktree, dryRun bool) (string, error) {
	if noWorktree {
		return repoPath, nil
	}
	if dryRun {
		return dispatchWorktreePath(ref.Repo, ref.Number)
	}
	return ensureDispatchWorktree(ctx, r, repoPath, ref)
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

	prompt := interactivePrompt(ref, issue)
	titleLine := interactiveTitleLine(ref, issue)
	queueDir := c.String("queue-dir")
	launchName := c.String("launch-name")
	channel := c.String("channel")
	surface := c.String("surface")
	repoPath, _ := localRepoPath(ref.Repo)
	noWorktree := c.Bool("no-worktree")

	url, err := dispatchURL(channel, surface, launchName)
	if err != nil {
		return fmt.Errorf("dispatch interactive: %w", err)
	}

	// Each dispatch lands in its own git worktree branched off main, so
	// concurrent dispatches into the same repo never collide on a shared
	// working tree (coilysiren/coily#285). --no-worktree opts out for the
	// rare case a session must work against the canonical checkout.
	// In dry-run we compute the path but never call git; in normal mode
	// we create the worktree (or reuse it if already present).
	cwd, err := resolveDispatchCwd(ctx, r, repoPath, ref, noWorktree, c.Bool("dry-run"))
	if err != nil {
		return fmt.Errorf("dispatch interactive: %w", err)
	}

	entry := dispatchQueueEntry{
		SchemaVersion: dispatchQueueSchemaVersion,
		Ref:           ref.String(),
		Title:         strings.TrimSpace(issue.Title),
		Cwd:           cwd,
		Prompt:        prompt,
	}

	if c.Bool("dry-run") {
		entryJSON, _ := json.MarshalIndent(entry, "", "  ")
		fmt.Printf("# dispatch interactive (dry-run)\n")
		fmt.Printf("issue:        %s\n", ref)
		fmt.Printf("url:          %s\n", issue.URL)
		fmt.Printf("cwd:          %s\n", cwd)
		fmt.Printf("queue-dir:    %s\n", queueDir)
		fmt.Printf("channel:      %s\n", channel)
		fmt.Printf("surface:      %s\n", surface)
		fmt.Printf("dispatch-url: %s\n", url)
		fmt.Printf("----- title -----\n%s\n----- end -----\n", titleLine)
		fmt.Printf("----- queue entry -----\n%s\n----- end -----\n", string(entryJSON))
		return nil
	}

	// Pre-trust the cwd so the dispatched session never stalls on the
	// folder-trust prompt. Soft-fail: a missed trust prime is a papercut,
	// not a reason to drop the dispatch (coilysiren/coily#290).
	if err := ensureClaudeFolderTrust(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "dispatch interactive: could not pre-trust %s in ~/.claude.json (%v); the dispatched session may show the folder-trust prompt.\n", cwd, err)
	}

	queuePath, err := writeDispatchQueueEntry(queueDir, entry)
	if err != nil {
		return fmt.Errorf("dispatch interactive: write queue entry under %s: %w", queueDir, err)
	}

	if err := openWarpLaunch(ctx, r, url); err != nil {
		printInteractiveFallback(ref, cwd, prompt, url, queuePath, err)
		return nil
	}
	return nil
}

// writeDispatchQueueEntry writes a single JSON file to dir named
// <unix-nanos>-<8hex>.json with mode 0600. Atomic via tmp-write +
// rename within the same directory: the shim must not see a partial
// JSON payload if it pops the file before the write returns. Returns
// the final path so the soft-fail message can name it.
func writeDispatchQueueEntry(dir string, entry dispatchQueueEntry) (string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("mkdir queue dir: %w", err)
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("marshal queue entry: %w", err)
	}
	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		return "", fmt.Errorf("rand suffix: %w", err)
	}
	name := fmt.Sprintf("%d-%s.json", time.Now().UnixNano(), hex.EncodeToString(suffix))
	finalPath := filepath.Join(dir, name)
	tmpPath := finalPath + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o600); err != nil {
		return "", fmt.Errorf("write tmp queue entry: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("rename tmp to final: %w", err)
	}
	return finalPath, nil
}

// printInteractiveFallback emits the exact command the operator can paste
// in a manually-opened tab when the Warp URL fire fails. Soft-fail by
// design: missing Warp shouldn't error the audit row, just inform the
// operator that the automated path didn't fire. The queue entry stays
// on disk so a retry can consume it.
func printInteractiveFallback(ref *issueRef, repoPath, prompt, url, queuePath string, openErr error) {
	fmt.Fprintf(os.Stderr, "dispatch interactive: %s did not fire (%v).\n", url, openErr)
	fmt.Fprintf(os.Stderr, "Queue entry left at %s for the shim to consume on retry.\n", queuePath)
	fmt.Fprintf(os.Stderr, "Or paste this in a new tab cwd'd to %s:\n\n", repoPath)
	fmt.Fprintf(os.Stderr, "  cd %s && claude %q\n\n", repoPath, prompt)
	fmt.Fprintf(os.Stderr, "Issue: %s\n", ref)
}
