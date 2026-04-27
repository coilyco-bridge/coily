package trello

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v3"
)

// fakeCmd builds the minimal cli.Command surface resolveDir needs (a single
// --dir string flag), parsed from argv. urfave/cli v3's Command zero-value
// won't return flag values without a Run() pass, so we drive Run with a no-op
// action and pull the resolved Command out.
func fakeCmd(t *testing.T, argv ...string) *cli.Command {
	t.Helper()
	var captured *cli.Command
	app := &cli.Command{
		Flags: []cli.Flag{&cli.StringFlag{Name: "dir"}},
		Action: func(_ context.Context, c *cli.Command) error {
			captured = c
			return nil
		},
	}
	if err := app.Run(context.Background(), append([]string{"trello-test"}, argv...)); err != nil {
		t.Fatalf("cli run: %v", err)
	}
	return captured
}

func TestResolveDir_FlagWins(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	t.Setenv(EnvDir, t.TempDir()) // env points elsewhere; flag should still win
	got, err := resolveDir(fakeCmd(t, "--dir", dir))
	if err != nil {
		t.Fatalf("resolveDir: %v", err)
	}
	if got != filepath.Clean(dir) {
		t.Errorf("got %q, want %q", got, dir)
	}
}

func TestResolveDir_EnvFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	t.Setenv(EnvDir, dir)
	got, err := resolveDir(fakeCmd(t))
	if err != nil {
		t.Fatalf("resolveDir: %v", err)
	}
	if got != filepath.Clean(dir) {
		t.Errorf("got %q, want %q", got, dir)
	}
}

func TestResolveDir_MissingPackageJSON(t *testing.T) {
	t.Setenv(EnvDir, t.TempDir()) // empty dir -> no package.json
	if _, err := resolveDir(fakeCmd(t)); err == nil {
		t.Error("expected error for missing package.json")
	}
}

func TestBuildUpdateScriptArgs_OnlySetFlagsForwarded(t *testing.T) {
	var got *cli.Command
	app := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "list"},
			&cli.StringFlag{Name: "label-on"},
			&cli.StringFlag{Name: "label-off"},
			&cli.StringFlag{Name: "comment"},
			&cli.StringFlag{Name: "name"},
			&cli.StringFlag{Name: "desc"},
			&cli.BoolFlag{Name: "close"},
			&cli.BoolFlag{Name: "reopen"},
		},
		Action: func(_ context.Context, c *cli.Command) error { got = c; return nil },
	}
	if err := app.Run(context.Background(), []string{"trello-test", "--list", "Active", "--comment", "hello", "abc123"}); err != nil {
		t.Fatalf("cli run: %v", err)
	}
	out := buildUpdateScriptArgs(got)
	want := []string{"abc123", "--list", "Active", "--comment", "hello"}
	if len(out) != len(want) {
		t.Fatalf("got %v, want %v", out, want)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Errorf("argv[%d] = %q, want %q (full: %v)", i, out[i], want[i], out)
		}
	}
}
