package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInteractivePrompt_Minimal pins the "Work on issue <ref>" shape from
// #270. The launch-config script greps the coilysiren/<repo>#<N> token out
// of this string to derive cwd, so the format is a contract, not cosmetic.
func TestInteractivePrompt_Minimal(t *testing.T) {
	ref := &issueRef{Owner: "coilysiren", Repo: "coily", Number: 270}
	got := interactivePrompt(ref)
	want := "Work on issue coilysiren/coily#270"
	if got != want {
		t.Errorf("interactivePrompt = %q, want %q", got, want)
	}
	// Sanity: no preamble, no URL, no flair.
	for _, bad := range []string{"\n", "URL", "Title", "AGENTS", "--no-verify"} {
		if strings.Contains(got, bad) {
			t.Errorf("interactivePrompt should not contain %q, got %q", bad, got)
		}
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
