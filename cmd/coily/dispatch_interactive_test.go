package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInteractivePrompt_RefAndFirstAction pins the contract from #270 +
// #279. The shim greps the coilysiren/<repo>#<N> token out of the JSON
// payload via jq, but the prompt body itself is what claude sees, so
// the ref staying in the first sentence is a human-readability claim
// rather than a parsing claim. The first-action instruction lands in
// the same prompt because the agent otherwise skips the explicit issue
// fetch and works from the bare ref line, missing body, comments, and
// labels.
func TestInteractivePrompt_RefAndFirstAction(t *testing.T) {
	ref := &issueRef{Owner: "coilysiren", Repo: "coily", Number: 270}
	issue := &ghIssue{
		Number: 270,
		Title:  "split dispatch into headless/interactive",
		URL:    "https://github.com/coilysiren/coily/issues/270",
		State:  "open",
	}
	got := interactivePrompt(ref, issue, false)

	if !strings.HasPrefix(got, "Work on issue coilysiren/coily#270.") {
		t.Errorf("interactivePrompt prefix = %q, want \"Work on issue coilysiren/coily#270.\" lead", got)
	}
	for _, want := range []string{
		"First action",
		"coily ops gh issue view",
		"https://github.com/coilysiren/coily/issues/270",
		"--comments",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("interactivePrompt missing %q, got %q", want, got)
		}
	}
}

// TestInteractivePrompt_MergeBack pins #300: in worktree mode the prompt
// must tell the dispatched agent to land its branch on main itself,
// otherwise the dispatch/issue-N branch sits unmerged forever. The
// --no-worktree variant runs in the bare checkout on main, so it must
// NOT carry the merge-back paragraph.
func TestInteractivePrompt_MergeBack(t *testing.T) {
	ref := &issueRef{Owner: "coilysiren", Repo: "coily", Number: 300}
	issue := &ghIssue{
		Number: 300,
		Title:  "close the worktree lifecycle",
		URL:    "https://github.com/coilysiren/coily/issues/300",
		State:  "open",
	}

	withWorktree := interactivePrompt(ref, issue, false)
	for _, want := range []string{
		"dispatch/issue-300",
		"merge that branch into `main`",
		"push origin main",
		"Never force-push",
	} {
		if !strings.Contains(withWorktree, want) {
			t.Errorf("worktree-mode prompt missing %q, got %q", want, withWorktree)
		}
	}

	noWorktree := interactivePrompt(ref, issue, true)
	if strings.Contains(noWorktree, "force-push") || strings.Contains(noWorktree, "\n") {
		t.Errorf("--no-worktree prompt must stay single-line with no merge-back, got %q", noWorktree)
	}
}

// TestInteractiveTitleLine pins the self-identifying header shape coily
// embeds in the queue entry's title field and the shim echoes in the
// tab. Format is "<ref>: <title>", whitespace-trimmed (#279).
func TestInteractiveTitleLine(t *testing.T) {
	ref := &issueRef{Owner: "coilysiren", Repo: "coily", Number: 279}
	issue := &ghIssue{
		Number: 279,
		Title:  "  dispatch interactive: echo issue title and prime first agent action  ",
		URL:    "https://github.com/coilysiren/coily/issues/279",
		State:  "open",
	}
	got := interactiveTitleLine(ref, issue)
	want := "coilysiren/coily#279: dispatch interactive: echo issue title and prime first agent action"
	if got != want {
		t.Errorf("interactiveTitleLine = %q, want %q", got, want)
	}
}

// TestWriteDispatchQueueEntry_ModeAndJSON verifies the queue entry is
// written under the queue dir with mode 0600, named
// <unix-nanos>-<8hex>.json, and parseable as the shim's JSON schema.
func TestWriteDispatchQueueEntry_ModeAndJSON(t *testing.T) {
	dir := t.TempDir()
	entry := dispatchQueueEntry{
		SchemaVersion: dispatchQueueSchemaVersion,
		Ref:           "coilysiren/coily#280",
		Title:         "concurrency race on scratch path",
		Cwd:           "/Users/kai/projects/coilysiren/coily",
		Prompt:        "Work on issue coilysiren/coily#280. First action: ...",
	}
	path, err := writeDispatchQueueEntry(dir, entry)
	if err != nil {
		t.Fatalf("writeDispatchQueueEntry: %v", err)
	}

	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat queue entry: %v", err)
	}
	if got, want := st.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Errorf("queue entry mode = %o, want %o", got, want)
	}
	if !strings.HasSuffix(path, ".json") {
		t.Errorf("queue entry path = %q, want .json suffix", path)
	}
	if filepath.Dir(path) != dir {
		t.Errorf("queue entry dir = %q, want %q", filepath.Dir(path), dir)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read queue entry: %v", err)
	}
	var got dispatchQueueEntry
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal queue entry: %v\nbody: %s", err, b)
	}
	if got != entry {
		t.Errorf("queue entry roundtrip = %+v, want %+v", got, entry)
	}
}

// TestWriteDispatchQueueEntry_UniqueFilenames verifies two back-to-back
// writes produce distinct filenames so concurrent dispatches never
// collide on the same queue path. Pins the singleton-fix from #280.
func TestWriteDispatchQueueEntry_UniqueFilenames(t *testing.T) {
	dir := t.TempDir()
	entry := dispatchQueueEntry{
		SchemaVersion: dispatchQueueSchemaVersion,
		Ref:           "coilysiren/coily#280",
		Title:         "concurrency race on scratch path",
		Cwd:           "/Users/kai/projects/coilysiren/coily",
		Prompt:        "Work on issue coilysiren/coily#280.",
	}
	seen := map[string]bool{}
	for i := 0; i < 16; i++ {
		path, err := writeDispatchQueueEntry(dir, entry)
		if err != nil {
			t.Fatalf("writeDispatchQueueEntry iteration %d: %v", i, err)
		}
		if seen[path] {
			t.Errorf("duplicate queue path %q at iteration %d", path, i)
		}
		seen[path] = true
	}
}

// TestDispatchInteractiveDefaults pins the seam strings between coily and
// the agentic-os shim. Changing either side without the other breaks
// the contract.
func TestDispatchInteractiveDefaults(t *testing.T) {
	if defaultDispatchQueueDir != "/tmp/coily-dispatch-queue" {
		t.Errorf("defaultDispatchQueueDir = %q, want /tmp/coily-dispatch-queue (FIFO seam consumed by claude-dispatch-interactive.sh per coilysiren/coily#280)", defaultDispatchQueueDir)
	}
	if defaultDispatchLaunchName != "claude-dispatch-interactive" {
		t.Errorf("defaultDispatchLaunchName = %q, want claude-dispatch-interactive", defaultDispatchLaunchName)
	}
	if defaultDispatchChannel != "preview" {
		t.Errorf("defaultDispatchChannel = %q, want preview (Preview is the Mac daily driver per coilysiren/agentic-os#107)", defaultDispatchChannel)
	}
	if defaultDispatchSurface != "tab" {
		t.Errorf("defaultDispatchSurface = %q, want tab (tab_config opens a new tab via warpdotdev/Warp#9379, per coilysiren/coily#274)", defaultDispatchSurface)
	}
	if dispatchQueueSchemaVersion != 1 {
		t.Errorf("dispatchQueueSchemaVersion = %d, want 1 (schema version pin so the shim can reject unknown versions cleanly)", dispatchQueueSchemaVersion)
	}
}

// TestDispatchURL_ChannelSurfaceMatrix pins the four URL shapes coily
// can produce: (preview, stable) × (tab, window). Stable always lands at
// warp://, Preview always at warppreview://; tab routes via tab_config/,
// window routes via launch/. No LaunchServices toggle flips channel; no
// build-date sniff flips surface.
func TestDispatchURL_ChannelSurfaceMatrix(t *testing.T) {
	cases := []struct {
		channel string
		surface string
		want    string
	}{
		{"preview", "tab", "warppreview://tab_config/claude-dispatch-interactive"},
		{"preview", "window", "warppreview://launch/claude-dispatch-interactive"},
		{"stable", "tab", "warp://tab_config/claude-dispatch-interactive"},
		{"stable", "window", "warp://launch/claude-dispatch-interactive"},
	}
	for _, tc := range cases {
		got, err := dispatchURL(tc.channel, tc.surface, "claude-dispatch-interactive")
		if err != nil {
			t.Errorf("dispatchURL(%q,%q): unexpected err: %v", tc.channel, tc.surface, err)
			continue
		}
		if got != tc.want {
			t.Errorf("dispatchURL(%q,%q) = %q, want %q", tc.channel, tc.surface, got, tc.want)
		}
	}
}

// TestDispatchURL_RejectsUnknownChannel pins the "preview | stable"
// gate. An unknown channel must error rather than silently fall through
// to a default scheme, since picking the wrong channel opens the wrong
// app.
func TestDispatchURL_RejectsUnknownChannel(t *testing.T) {
	_, err := dispatchURL("garbage", "tab", "claude-dispatch-interactive")
	if err == nil {
		t.Fatal("dispatchURL(garbage,tab) should error, got nil")
	}
	msg := err.Error()
	for _, want := range []string{"preview", "stable", "invalid"} {
		if !strings.Contains(msg, want) {
			t.Errorf("dispatchURL(garbage,tab) error = %q, want substring %q", msg, want)
		}
	}
}

// TestDispatchURL_RejectsUnknownSurface pins the "tab | window" gate.
// An unknown surface must error rather than silently fall through to a
// default path, since the URI paths route to different Warp behaviors
// (tab_config = new tab, launch = new window).
func TestDispatchURL_RejectsUnknownSurface(t *testing.T) {
	_, err := dispatchURL("preview", "garbage", "claude-dispatch-interactive")
	if err == nil {
		t.Fatal("dispatchURL(preview,garbage) should error, got nil")
	}
	msg := err.Error()
	for _, want := range []string{"tab", "window", "invalid"} {
		if !strings.Contains(msg, want) {
			t.Errorf("dispatchURL(preview,garbage) error = %q, want substring %q", msg, want)
		}
	}
}

// TestDispatchBare_ErrorsWithModeGate pins #270's "no default mode" rule.
// Bare `coily dispatch <ref>` must error and name the two valid modes; it
// must not silently fall through to either headless or interactive.
func TestDispatchBare_ErrorsWithModeGate(t *testing.T) {
	r := newTestRunner(t)
	cmd := r.dispatchCommand()
	err := cmd.Run(context.Background(), []string{"dispatch", "coilysiren/coily#270"})
	if err == nil {
		t.Fatal("bare dispatch <ref> should error with mode-gate, got nil")
	}
	msg := err.Error()
	for _, want := range []string{"specify mode", "interactive", "headless"} {
		if !strings.Contains(msg, want) {
			t.Errorf("dispatch bare error = %q, want substring %q", msg, want)
		}
	}
}

// TestDispatchHasModeSubverbs proves the headless + interactive subverbs
// hang off the dispatch parent. Catches a refactor that accidentally
// drops one (e.g., removing a builder method from the parent's Commands).
func TestDispatchHasModeSubverbs(t *testing.T) {
	r := newTestRunner(t)
	cmd := r.dispatchCommand()
	got := map[string]bool{}
	for _, sub := range cmd.Commands {
		got[sub.Name] = true
	}
	for _, want := range []string{"headless", "interactive"} {
		if !got[want] {
			t.Errorf("dispatch parent missing subverb %q (got %v)", want, got)
		}
	}
}

// TestDispatchWorktreePath pins the layout
// ~/projects/coilysiren/.dispatch-worktrees/<repo>/issue-<N>. Lives
// outside any repo so no per-repo .gitignore churn (coilysiren/coily#285).
func TestDispatchWorktreePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	got, err := dispatchWorktreePath("coily", 285)
	if err != nil {
		t.Fatalf("dispatchWorktreePath: %v", err)
	}
	want := filepath.Join(home, "projects", "coilysiren", ".dispatch-worktrees", "coily", "issue-285")
	if got != want {
		t.Errorf("dispatchWorktreePath(coily,285) = %q, want %q", got, want)
	}
}

// TestDispatchWorktreeBranch pins the branch name shape
// `dispatch/issue-<N>`. Predictable so re-dispatching the same issue
// reuses the same branch (idempotency contract from #285).
func TestDispatchWorktreeBranch(t *testing.T) {
	if got, want := dispatchWorktreeBranch(285), "dispatch/issue-285"; got != want {
		t.Errorf("dispatchWorktreeBranch(285) = %q, want %q", got, want)
	}
}

// TestEnsureDispatchWorktree_CallsGit verifies the production path:
// when no worktree exists at the target path, ensure runs
// `git -C <repoPath> worktree add -B <branch> <worktreePath>` exactly
// once with the expected arguments.
func TestEnsureDispatchWorktree_CallsGit(t *testing.T) {
	r := newTestRunner(t)
	ref := &issueRef{Owner: "coilysiren", Repo: "coily", Number: 285}
	repoPath := t.TempDir()

	prevRoot := dispatchWorktreeRootOverride
	dispatchWorktreeRootOverride = t.TempDir()
	t.Cleanup(func() { dispatchWorktreeRootOverride = prevRoot })

	var gotRepo, gotBranch, gotPath string
	calls := 0
	prev := runWorktreeAdd
	runWorktreeAdd = func(_ context.Context, _ *Runner, gitDir, gitBranch, gitWT string) error {
		calls++
		gotRepo, gotBranch, gotPath = gitDir, gitBranch, gitWT
		return nil
	}
	t.Cleanup(func() { runWorktreeAdd = prev })

	wt, err := ensureDispatchWorktree(context.Background(), r, repoPath, ref)
	if err != nil {
		t.Fatalf("ensureDispatchWorktree: %v", err)
	}
	if calls != 1 {
		t.Errorf("runWorktreeAdd called %d times, want 1", calls)
	}
	if gotRepo != repoPath {
		t.Errorf("git -C dir = %q, want %q", gotRepo, repoPath)
	}
	if gotBranch != "dispatch/issue-285" {
		t.Errorf("branch = %q, want dispatch/issue-285", gotBranch)
	}
	if !strings.HasSuffix(gotPath, filepath.Join("coily", "issue-285")) {
		t.Errorf("worktree path = %q, want suffix coily/issue-285", gotPath)
	}
	if wt != gotPath {
		t.Errorf("ensure returned %q, runWorktreeAdd saw %q", wt, gotPath)
	}
}

// TestEnsureDispatchWorktree_Idempotent verifies the reuse path: when a
// .git entry already exists under the target worktree path, ensure
// returns the path without calling runWorktreeAdd. Reason: re-dispatching
// the same issue must land in the same worktree rather than fail or
// proliferate (idempotency contract from #285).
func TestEnsureDispatchWorktree_Idempotent(t *testing.T) {
	r := newTestRunner(t)
	ref := &issueRef{Owner: "coilysiren", Repo: "coily", Number: 285}
	repoPath := t.TempDir()

	root := t.TempDir()
	prevRoot := dispatchWorktreeRootOverride
	dispatchWorktreeRootOverride = root
	t.Cleanup(func() { dispatchWorktreeRootOverride = prevRoot })

	// Simulate an existing worktree: <root>/coily/issue-285/.git
	existing := filepath.Join(root, "coily", "issue-285")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatalf("mkdir existing worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(existing, ".git"), []byte("gitdir: /elsewhere\n"), 0o644); err != nil {
		t.Fatalf("write .git: %v", err)
	}

	calls := 0
	prev := runWorktreeAdd
	runWorktreeAdd = func(_ context.Context, _ *Runner, _, _, _ string) error {
		calls++
		return nil
	}
	t.Cleanup(func() { runWorktreeAdd = prev })

	wt, err := ensureDispatchWorktree(context.Background(), r, repoPath, ref)
	if err != nil {
		t.Fatalf("ensureDispatchWorktree: %v", err)
	}
	if calls != 0 {
		t.Errorf("runWorktreeAdd called %d times, want 0 (existing worktree must be reused)", calls)
	}
	if wt != existing {
		t.Errorf("ensure returned %q, want %q", wt, existing)
	}
}
