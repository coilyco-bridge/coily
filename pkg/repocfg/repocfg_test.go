package repocfg_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/coilysiren/coily/pkg/repocfg"
)

func writeConfig(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, repocfg.Filename)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoad_StringForm(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `
commands:
  test: go test ./...
  lint: golangci-lint run ./...
`)
	cfg, err := repocfg.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Path != path {
		t.Errorf("Path = %q, want %q", cfg.Path, path)
	}
	if got := len(cfg.Commands); got != 2 {
		t.Fatalf("got %d commands, want 2", got)
	}
	// Commands are sorted by name. "lint" < "test".
	if cfg.Commands[0].Name != "lint" || cfg.Commands[1].Name != "test" {
		t.Errorf("order = [%s, %s], want [lint, test]", cfg.Commands[0].Name, cfg.Commands[1].Name)
	}
	want := []string{"go", "test", "./..."}
	got := cfg.Commands[1].Argv
	if len(got) != len(want) {
		t.Fatalf("test argv = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("test argv[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLoad_MappingForm(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `
commands:
  test:
    run: go test ./...
    description: Run the full unit suite.
`)
	cfg, err := repocfg.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	c := cfg.Commands[0]
	if c.Description != "Run the full unit suite." {
		t.Errorf("Description = %q", c.Description)
	}
}

func TestLoad_RejectsShellMetacharacter(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `
commands:
  bad: echo hi; rm -rf /tmp/foo
`)
	if _, err := repocfg.Load(path); err == nil {
		t.Error("Load accepted a command with a shell metacharacter")
	}
}

func TestLoad_RejectsPipeRedirect(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `
commands:
  bad: cat file | grep foo
`)
	if _, err := repocfg.Load(path); err == nil {
		t.Error("Load accepted a piped command")
	}
}

func TestLoad_RejectsEmptyRun(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `
commands:
  empty: ""
`)
	if _, err := repocfg.Load(path); err == nil {
		t.Error("Load accepted an empty run value")
	}
}

func TestLoad_RejectsIllegalName(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `
commands:
  "--flag": go test
`)
	if _, err := repocfg.Load(path); err == nil {
		t.Error("Load accepted a command name beginning with -")
	}
}

func TestDiscover_FindsInParentOverlay(t *testing.T) {
	// Discover prefers ./.coily/coily.yaml. Place the file under the overlay
	// directory at root and walk from a deep child.
	root := t.TempDir()
	overlay := filepath.Join(root, repocfg.LocalDirName)
	if err := os.MkdirAll(overlay, 0o700); err != nil {
		t.Fatalf("mkdir overlay: %v", err)
	}
	writeConfig(t, overlay, "commands: {test: go test ./...}\n")
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path, err := repocfg.Discover(deep)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	want := filepath.Join(overlay, repocfg.Filename)
	// Compare against evaluated symlinks because macOS TempDir returns /var,
	// which resolves to /private/var.
	gotR, _ := filepath.EvalSymlinks(path)
	wantR, _ := filepath.EvalSymlinks(want)
	if gotR != wantR {
		t.Errorf("Discover = %q, want %q", path, want)
	}
}

func TestDiscover_RejectsLegacyRootLocation(t *testing.T) {
	// A coily.yaml at the repo root (no .coily/ overlay) used to be the
	// canonical location. Now it's an error pointing at the new home.
	root := t.TempDir()
	writeConfig(t, root, "commands: {test: go test ./...}\n")
	_, err := repocfg.Discover(root)
	if !errors.Is(err, repocfg.ErrLegacyLocation) {
		t.Errorf("err = %v, want ErrLegacyLocation", err)
	}
}

func TestDiscover_OverlayWinsOverLegacy(t *testing.T) {
	// If both exist (during a partial migration), the overlay takes
	// precedence and the legacy file is ignored.
	root := t.TempDir()
	overlay := filepath.Join(root, repocfg.LocalDirName)
	if err := os.MkdirAll(overlay, 0o700); err != nil {
		t.Fatalf("mkdir overlay: %v", err)
	}
	writeConfig(t, overlay, "commands: {modern: go version}\n")
	writeConfig(t, root, "commands: {legacy: echo nope}\n")
	path, err := repocfg.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	want := filepath.Join(overlay, repocfg.Filename)
	gotR, _ := filepath.EvalSymlinks(path)
	wantR, _ := filepath.EvalSymlinks(want)
	if gotR != wantR {
		t.Errorf("Discover = %q, want %q", path, want)
	}
}

func TestDiscover_ReturnsErrNoConfig(t *testing.T) {
	dir := t.TempDir()
	_, err := repocfg.Discover(dir)
	if !errors.Is(err, repocfg.ErrNoConfig) {
		t.Errorf("err = %v, want ErrNoConfig", err)
	}
}

func TestDiscoverChildren_FindsOverlayInChild(t *testing.T) {
	// Layout: /parent/child/.coily/coily.yaml. Discovery from parent finds it.
	parent := t.TempDir()
	childOverlay := filepath.Join(parent, "child", repocfg.LocalDirName)
	if err := os.MkdirAll(childOverlay, 0o700); err != nil {
		t.Fatalf("mkdir child overlay: %v", err)
	}
	writeConfig(t, childOverlay, "commands: {test: go test ./...}\n")
	configs, err := repocfg.DiscoverChildren(parent)
	if err != nil {
		t.Fatalf("DiscoverChildren: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("len(configs) = %d, want 1", len(configs))
	}
	if configs[0].Commands[0].Name != "test" {
		t.Errorf("got %q, want test", configs[0].Commands[0].Name)
	}
}

func TestDiscoverChildren_SkipsLegacyRootForm(t *testing.T) {
	// A legacy /parent/child/coily.yaml (no .coily/ overlay) is intentionally
	// ignored. Child discovery is opt-in via the .coily/ overlay so unrelated
	// repos that happen to predate the migration don't get pulled in.
	parent := t.TempDir()
	childRoot := filepath.Join(parent, "legacy-child")
	if err := os.MkdirAll(childRoot, 0o700); err != nil {
		t.Fatalf("mkdir legacy child: %v", err)
	}
	writeConfig(t, childRoot, "commands: {test: go test}\n")
	configs, err := repocfg.DiscoverChildren(parent)
	if err != nil {
		t.Fatalf("DiscoverChildren: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("len(configs) = %d, want 0 (legacy form must be ignored)", len(configs))
	}
}

func TestDiscoverChildren_SkipsHiddenAndUnconfiguredChildren(t *testing.T) {
	// Hidden entries (.git, .vscode) are skipped. Children without a
	// .coily/coily.yaml are skipped. Files at parent level are skipped.
	parent := t.TempDir()
	if err := os.MkdirAll(filepath.Join(parent, ".git"), 0o700); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(parent, "no-config"), 0o700); err != nil {
		t.Fatalf("mkdir no-config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(parent, "stray-file.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	configs, err := repocfg.DiscoverChildren(parent)
	if err != nil {
		t.Fatalf("DiscoverChildren: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("len(configs) = %d, want 0", len(configs))
	}
}

func TestDiscoverChildren_SkipsMalformedChild(t *testing.T) {
	// A child whose coily.yaml fails to parse must not abort the whole scan.
	// The good child is still returned.
	parent := t.TempDir()
	bad := filepath.Join(parent, "bad", repocfg.LocalDirName)
	good := filepath.Join(parent, "good", repocfg.LocalDirName)
	if err := os.MkdirAll(bad, 0o700); err != nil {
		t.Fatalf("mkdir bad: %v", err)
	}
	if err := os.MkdirAll(good, 0o700); err != nil {
		t.Fatalf("mkdir good: %v", err)
	}
	writeConfig(t, bad, "commands: {oops: 'echo hi; rm -rf /'}\n")
	writeConfig(t, good, "commands: {test: go test}\n")
	configs, err := repocfg.DiscoverChildren(parent)
	if err != nil {
		t.Fatalf("DiscoverChildren: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("len(configs) = %d, want 1 (bad child must be silently skipped)", len(configs))
	}
	if configs[0].Commands[0].Name != "test" {
		t.Errorf("got %q, want test", configs[0].Commands[0].Name)
	}
}

func TestDiscoverChildren_SortedByPath(t *testing.T) {
	parent := t.TempDir()
	for _, name := range []string{"zebra", "apple", "mango"} {
		dir := filepath.Join(parent, name, repocfg.LocalDirName)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		writeConfig(t, dir, "commands: {test: go test}\n")
	}
	configs, err := repocfg.DiscoverChildren(parent)
	if err != nil {
		t.Fatalf("DiscoverChildren: %v", err)
	}
	if len(configs) != 3 {
		t.Fatalf("len(configs) = %d, want 3", len(configs))
	}
	for i := 1; i < len(configs); i++ {
		if configs[i-1].Path >= configs[i].Path {
			t.Errorf("configs not sorted: %s >= %s", configs[i-1].Path, configs[i].Path)
		}
	}
}

func TestLoadDefault_UsesEnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, "commands: {test: go test}\n")
	t.Setenv(repocfg.EnvOverride, path)
	cfg, err := repocfg.LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault: %v", err)
	}
	if cfg.Commands[0].Name != "test" {
		t.Errorf("got %q, want test", cfg.Commands[0].Name)
	}
}
