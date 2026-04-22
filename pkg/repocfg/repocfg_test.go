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

func TestDiscover_FindsInParent(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "commands: {test: go test ./...}\n")
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path, err := repocfg.Discover(deep)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	want := filepath.Join(root, repocfg.Filename)
	// Compare against evaluated symlinks because macOS TempDir returns /var,
	// which resolves to /private/var.
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
