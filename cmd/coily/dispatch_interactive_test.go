package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInteractivePrompt_RefAndFirstAction pins the contract from #270 +
// #279. The launch-config script greps the coilysiren/<repo>#<N> token
// out of this string to derive cwd, so the ref stays in the first
// sentence. The first-action instruction lands in the same prompt
// because the agent otherwise skips the explicit issue fetch and works
// from the bare ref line, missing body, comments, and labels.
func TestInteractivePrompt_RefAndFirstAction(t *testing.T) {
	ref := &issueRef{Owner: "coilysiren", Repo: "coily", Number: 270}
	issue := &ghIssue{
		Number: 270,
		Title:  "split dispatch into headless/interactive",
		URL:    "https://github.com/coilysiren/coily/issues/270",
		State:  "open",
	}
	got := interactivePrompt(ref, issue)

	// Ref stays in the first sentence so the shim's grep keeps working.
	if !strings.HasPrefix(got, "Work on issue coilysiren/coily#270.") {
		t.Errorf("interactivePrompt prefix = %q, want \"Work on issue coilysiren/coily#270.\" lead", got)
	}
	// First-action instruction primes `coily ops gh issue view` with the
	// resolved URL form (the owner/repo#N form is not accepted by gh).
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
	// Single line: keeps the shim's PROMPT="$(cat ...)" round-trip
	// trivial and the prompt readable in the Warp tab header.
	if strings.Contains(got, "\n") {
		t.Errorf("interactivePrompt should be single-line, got %q", got)
	}
}

// TestInteractiveTitleLine pins the self-identifying header shape coily
// writes to the title sidecar and the shim echoes in the tab. Format is
// "<ref>: <title>", whitespace-trimmed (#279).
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

// TestWriteDispatchScratchFile_ModeAndContents verifies the scratch file
// is written with mode 0600 (only the running user can read it) and
// carries the prompt body verbatim plus a trailing newline.
func TestWriteDispatchScratchFile_ModeAndContents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.txt")
	prompt := "Work on issue coilysiren/coily#270"

	if err := writeDispatchScratchFile(path, prompt); err != nil {
		t.Fatalf("writeDispatchScratchFile: %v", err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat scratch: %v", err)
	}
	if got, want := st.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Errorf("scratch mode = %o, want %o", got, want)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read scratch: %v", err)
	}
	if got, want := string(b), prompt+"\n"; got != want {
		t.Errorf("scratch contents = %q, want %q", got, want)
	}
}

// TestDispatchInteractiveDefaults pins the seam strings between coily and
// the agentic-os launch config. Changing either side without the other
// breaks the contract.
func TestDispatchInteractiveDefaults(t *testing.T) {
	if defaultDispatchScratchPath != "/tmp/coily-dispatch-prompt.txt" {
		t.Errorf("defaultDispatchScratchPath = %q, want /tmp/coily-dispatch-prompt.txt", defaultDispatchScratchPath)
	}
	if defaultDispatchTitlePath != "/tmp/coily-dispatch-title.txt" {
		t.Errorf("defaultDispatchTitlePath = %q, want /tmp/coily-dispatch-title.txt (sidecar seam consumed by claude-dispatch-interactive.sh per coilysiren/coily#279)", defaultDispatchTitlePath)
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
