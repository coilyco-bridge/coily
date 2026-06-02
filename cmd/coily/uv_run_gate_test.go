package main

import "testing"

const testRepoRoot = "/Users/kai/projects/coilyco-bridge/coily"

func TestUvRunPathViolation_Blocked(t *testing.T) {
	cases := [][]string{
		// Scratch-tier absolute paths - the filed shape (coily#10).
		{"run", "/tmp/jobs_failure2.py"},
		{"run", "/tmp/step_log.py"},
		{"run", "/var/tmp/x.py"},
		{"run", "/dev/shm/x.py"},
		{"run", "/private/tmp/x.py"},
		// Absolute path outside the repo root.
		{"run", "/Users/kai/evil.py"},
		// Relative path that climbs out of the repo.
		{"run", "../../../../tmp/x.py"},
		// Scratch path behind a uv flag.
		{"run", "--with", "requests", "/tmp/x.py"},
		// Interpreter target plus a scratch script (uv run python /tmp/x.py).
		{"run", "python", "/tmp/x.py"},
		// `uv tool run` carries the same hole.
		{"tool", "run", "/tmp/x.py"},
	}
	for _, argv := range cases {
		if _, blocked := uvRunPathViolation(argv, testRepoRoot, testRepoRoot); !blocked {
			t.Errorf("uvRunPathViolation(%v) = not blocked, want blocked", argv)
		}
	}
}

func TestUvRunPathViolation_Allowed(t *testing.T) {
	cases := [][]string{
		// Bare tool / module names.
		{"run", "pytest"},
		{"run", "ruff", "check"},
		{"run", "pre-commit", "run", "-a"},
		// Project-rooted relative script paths.
		{"run", "script.py"},
		{"run", "./script.py"},
		{"run", "tests/foo.py"},
		{"run", "pytest", "tests/unit/foo.py"},
		// Absolute path INSIDE the repo root.
		{"run", testRepoRoot + "/scripts/dev.py"},
		// Non-run uv verbs coily itself relies on: never gated.
		{"tool", "install", "pre-commit", "--with", "pre-commit-uv"},
		{"pip", "install", "-r", "/tmp/requirements.txt"},
		{"sync"},
		{"add", "requests"},
		// Empty / trivial.
		{},
		{"run"},
	}
	for _, argv := range cases {
		if _, blocked := uvRunPathViolation(argv, testRepoRoot, testRepoRoot); blocked {
			t.Errorf("uvRunPathViolation(%v) = blocked, want allowed", argv)
		}
	}
}

// An absolute script path under a different repo root escapes and blocks;
// a bare tool name passes regardless of root.
func TestUvRunPathViolation_AbsoluteOutsideRoot(t *testing.T) {
	const otherRoot = "/Users/kai/projects/other-repo"
	if _, blocked := uvRunPathViolation([]string{"run", "/opt/thing/x.py"}, otherRoot, otherRoot); !blocked {
		t.Error("absolute path outside the repo root should block")
	}
	if _, blocked := uvRunPathViolation([]string{"run", "pytest"}, otherRoot, otherRoot); blocked {
		t.Error("bare tool name should pass")
	}
}

func TestUvPreflightGate_BlocksScratchRun(t *testing.T) {
	// Integration-ish: the wired gate returns a hard error for the filed
	// shape. cwd/repoRoot resolution happens inside; a scratch-tier
	// absolute path blocks regardless of where the process runs.
	if err := uvPreflightGate([]string{"run", "/tmp/jobs_failure2.py"}); err == nil {
		t.Error("uvPreflightGate(run /tmp/...) = nil, want hard error")
	}
	if err := uvPreflightGate([]string{"sync"}); err != nil {
		t.Errorf("uvPreflightGate(sync) = %v, want nil", err)
	}
}

func TestLooksLikePath(t *testing.T) {
	paths := []string{"/tmp/x.py", "./x", "../x", "~/x", "a/b", "foo.py", "FOO.PY", "x.sh"}
	for _, p := range paths {
		if !looksLikePath(p) {
			t.Errorf("looksLikePath(%q) = false, want true", p)
		}
	}
	names := []string{"pytest", "ruff", "mod", "pre-commit", ""}
	for _, n := range names {
		if looksLikePath(n) {
			t.Errorf("looksLikePath(%q) = true, want false", n)
		}
	}
}
