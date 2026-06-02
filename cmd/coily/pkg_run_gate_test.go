package main

import "testing"

const testRepoRoot = "/Users/kai/projects/coilyco-bridge/coily"

func TestPkgRunPathViolation_Blocked(t *testing.T) {
	cases := []struct {
		bin  string
		argv []string
	}{
		// uv - the filed shape (coily#10).
		{"uv", []string{"run", "/tmp/jobs_failure2.py"}},
		{"uv", []string{"run", "/var/tmp/x.py"}},
		{"uv", []string{"run", "/dev/shm/x.py"}},
		{"uv", []string{"run", "/private/tmp/x.py"}},
		{"uv", []string{"run", "/Users/kai/evil.py"}},   // absolute outside repo
		{"uv", []string{"run", "../../../../tmp/x.py"}}, // climbs out of repo
		{"uv", []string{"run", "--with", "requests", "/tmp/x.py"}},
		{"uv", []string{"run", "python", "/tmp/x.py"}}, // interpreter + scratch script
		{"uv", []string{"tool", "run", "/tmp/x.py"}},
		// bun - general-purpose runtime, bare-file exec and `run`.
		{"bun", []string{"/tmp/x.ts"}},
		{"bun", []string{"run", "/tmp/x.ts"}},
		{"bun", []string{"run", "--hot", "/tmp/x.ts"}},
		{"bun", []string{"--watch", "/tmp/x.js"}},
		// pipx run.
		{"pipx", []string{"run", "/tmp/x.py"}},
		// poetry run launders an interpreter + path.
		{"poetry", []string{"run", "python", "/tmp/x.py"}},
		// bundle exec.
		{"bundle", []string{"exec", "ruby", "/tmp/x.rb"}},
		// npm / pnpm / yarn exec + dlx.
		{"npm", []string{"exec", "--", "node", "/tmp/x.js"}},
		{"npm", []string{"x", "node", "/tmp/x.js"}},
		{"pnpm", []string{"exec", "node", "/tmp/x.js"}},
		{"pnpm", []string{"dlx", "/tmp/x.js"}},
		{"yarn", []string{"dlx", "/tmp/x.js"}},
		// nix run / eval / shell / develop against a scratch path.
		{"nix", []string{"run", "/tmp/flake"}},
		{"nix", []string{"eval", "-f", "/tmp/x.nix"}},
		{"nix", []string{"shell", "nixpkgs#x", "--command", "bash", "/tmp/y.sh"}},
	}
	for _, c := range cases {
		if _, blocked := pkgRunPathViolation(c.bin, c.argv, testRepoRoot, testRepoRoot); !blocked {
			t.Errorf("pkgRunPathViolation(%q, %v) = not blocked, want blocked", c.bin, c.argv)
		}
	}
}

func TestPkgRunPathViolation_Allowed(t *testing.T) {
	cases := []struct {
		bin  string
		argv []string
	}{
		// Bare tool / module names.
		{"uv", []string{"run", "pytest"}},
		{"uv", []string{"run", "ruff", "check"}},
		{"bun", []string{"run", "build"}},
		{"bun", []string{"test"}},
		{"pnpm", []string{"exec", "eslint", "src/"}},
		{"poetry", []string{"run", "pytest"}},
		{"bundle", []string{"exec", "rspec"}},
		{"nix", []string{"run", "nixpkgs#hello"}},
		{"nix", []string{"run", ".#myapp"}},
		// Project-rooted relative script paths.
		{"uv", []string{"run", "script.py"}},
		{"uv", []string{"run", "./script.py"}},
		{"uv", []string{"run", "pytest", "tests/unit/foo.py"}},
		{"bun", []string{"run", "src/index.ts"}},
		// Absolute path INSIDE the repo root.
		{"uv", []string{"run", testRepoRoot + "/scripts/dev.py"}},
		// Non-run / non-exec verbs: never gated even with a scratch path arg.
		{"uv", []string{"tool", "install", "pre-commit", "--with", "pre-commit-uv"}},
		{"uv", []string{"pip", "install", "-r", "/tmp/requirements.txt"}},
		{"uv", []string{"sync"}},
		{"npm", []string{"install", "/tmp/local.tgz"}},
		{"npm", []string{"run", "build"}}, // npm run is not gated (package.json scripts)
		{"pnpm", []string{"add", "/tmp/local"}},
		{"nix", []string{"build", "/tmp/flake"}}, // nix build is not a run/exec verb
		// Bins with no run/exec gate at all.
		{"pip", []string{"install", "/tmp/pkg"}},
		{"cargo", []string{"run", "--manifest-path", "/tmp/Cargo.toml"}},
		{"gem", []string{"install", "/tmp/x.gem"}},
		// Empty / trivial.
		{"uv", []string{}},
		{"uv", []string{"run"}},
		{"bun", []string{}},
	}
	for _, c := range cases {
		if _, blocked := pkgRunPathViolation(c.bin, c.argv, testRepoRoot, testRepoRoot); blocked {
			t.Errorf("pkgRunPathViolation(%q, %v) = blocked, want allowed", c.bin, c.argv)
		}
	}
}

// An absolute script path under a different repo root escapes and blocks; a
// bare tool name passes regardless of root.
func TestPkgRunPathViolation_AbsoluteOutsideRoot(t *testing.T) {
	const otherRoot = "/Users/kai/projects/other-repo"
	if _, blocked := pkgRunPathViolation("uv", []string{"run", "/opt/thing/x.py"}, otherRoot, otherRoot); !blocked {
		t.Error("absolute path outside the repo root should block")
	}
	if _, blocked := pkgRunPathViolation("uv", []string{"run", "pytest"}, otherRoot, otherRoot); blocked {
		t.Error("bare tool name should pass")
	}
}

// The wired gate returns a hard error for the filed shape and nil for a
// benign one. cwd/repoRoot resolution happens inside; a scratch-tier
// absolute path blocks regardless of where the process runs.
func TestPkgRunGate_WiredBehavior(t *testing.T) {
	if g := pkgRunGate("uv"); g == nil || g([]string{"run", "/tmp/x.py"}) == nil {
		t.Error("pkgRunGate(uv)(run /tmp/x.py) should hard-error")
	}
	if g := pkgRunGate("bun"); g == nil || g([]string{"/tmp/x.ts"}) == nil {
		t.Error("pkgRunGate(bun)(/tmp/x.ts) should hard-error")
	}
	if g := pkgRunGate("uv"); g == nil || g([]string{"sync"}) != nil {
		t.Error("pkgRunGate(uv)(sync) should pass")
	}
	// Unregistered bins get no gate.
	if g := pkgRunGate("pip"); g != nil {
		t.Error("pkgRunGate(pip) should be nil (ungated)")
	}
	if g := pkgRunGate("cargo"); g != nil {
		t.Error("pkgRunGate(cargo) should be nil (ungated)")
	}
}

// Every bin wired with pkgRunGate in ptPkg has a registry entry, and every
// registered bin is a real ptPkg entry. Catches dead config / no-op gates.
func TestPkgRunSpecs_MatchPtPkgWiring(t *testing.T) {
	pkgBins := map[string]bool{}
	for _, e := range ptPkg {
		pkgBins[e.Bin] = true
	}
	for bin := range pkgRunSpecs {
		if !pkgBins[bin] {
			t.Errorf("pkgRunSpecs has %q but ptPkg has no such entry", bin)
		}
	}
}

func TestLooksLikePath(t *testing.T) {
	paths := []string{"/tmp/x.py", "./x", "../x", "~/x", "a/b", "foo.py", "FOO.PY", "x.sh", "x.ts"}
	for _, p := range paths {
		if !looksLikePath(p) {
			t.Errorf("looksLikePath(%q) = false, want true", p)
		}
	}
	names := []string{"pytest", "ruff", "mod", "build", "nixpkgs#hello", ""}
	for _, n := range names {
		if looksLikePath(n) {
			t.Errorf("looksLikePath(%q) = true, want false", n)
		}
	}
}
