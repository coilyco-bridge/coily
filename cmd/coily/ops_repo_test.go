package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/repocfg"
	"github.com/urfave/cli/v3"
)

// TestBuildExecFromChildren_AggregatesCommands proves that when cwd has
// no ancestor coily.yaml, child discovery still yields a usable exec
// command tree: each unique command name across direct children becomes
// a subcommand, sorted by name.
func TestBuildExecFromChildren_AggregatesCommands(t *testing.T) {
	parent := t.TempDir()
	mkChild(t, parent, "alpha", "commands:\n  alpha-only: true\n  shared: true\n")
	mkChild(t, parent, "beta", "commands:\n  beta-only: true\n  shared: true\n")

	cfgs, err := repocfg.DiscoverChildren(parent)
	if err != nil {
		t.Fatalf("DiscoverChildren: %v", err)
	}
	r := newTestRunner(t)
	exec := r.buildExecFromChildren(cfgs)

	want := []string{"alpha-only", "beta-only", "shared"}
	got := make([]string, 0, len(exec.Commands))
	for _, c := range exec.Commands {
		got = append(got, c.Name)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("subcommand names = %v, want %v", got, want)
	}
}

// TestBuildExecFromChildren_AmbiguousErrors proves that a command name
// declared by multiple direct children produces an error subcommand
// listing the matches, rather than picking one silently.
func TestBuildExecFromChildren_AmbiguousErrors(t *testing.T) {
	parent := t.TempDir()
	mkChild(t, parent, "alpha", "commands:\n  shared: true\n")
	mkChild(t, parent, "beta", "commands:\n  shared: true\n")

	cfgs, err := repocfg.DiscoverChildren(parent)
	if err != nil {
		t.Fatalf("DiscoverChildren: %v", err)
	}
	r := newTestRunner(t)
	root := wrapExecRoot(r.buildExecFromChildren(cfgs))
	err = root.Run(t.Context(), []string{"coily", "exec", "shared"})
	if err == nil {
		t.Fatal("ambiguous command did not error")
	}
	if k := errKind(err); k != "exec_ambiguous_children" {
		t.Errorf("err kind = %q, want exec_ambiguous_children (err=%v)", k, err)
	}
	if !strings.Contains(err.Error(), "alpha") || !strings.Contains(err.Error(), "beta") {
		t.Errorf("error message %q does not name both children", err.Error())
	}
}

// TestBuildChildRepoCommand_BindsAuditScopeToChild is the headline-case
// invariant: running `coily exec daily-social` from above coilyco-ai
// (the issue's example) must bind the audit row's commit-scope to the
// matched child repo, not cwd's git toplevel. That binding is what
// keeps repo-scoped audit queries (`coily git audit-show`) working
// when the operator runs one level above the target.
func TestBuildChildRepoCommand_BindsAuditScopeToChild(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	if _, err := exec.LookPath("true"); err != nil {
		t.Skip("true not on PATH")
	}

	repoRoot := initSecurityClaimRepo(t)
	cfgDir := filepath.Join(repoRoot, ".coily")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "coily.yaml")
	if err := os.WriteFile(cfgPath, []byte("commands:\n  noop: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGitForClaim(t, repoRoot, "add", ".coily/coily.yaml")
	mustGitForClaim(t, repoRoot, "commit", "-m", "add coily.yaml")
	mustGitForClaim(t, repoRoot, "push")

	cfg, err := repocfg.Load(cfgPath)
	if err != nil {
		t.Fatalf("repocfg.Load: %v", err)
	}

	r := newSecurityClaimRunnerWithAudit(t)
	cmd := r.buildChildRepoCommand(cfg, cfg.Commands[0])
	root := wrapExecRoot(cmd)
	if err := root.Run(t.Context(), []string{"coily", "noop"}); err != nil {
		t.Fatalf("noop run: %v", err)
	}

	rec := lastAuditRecord(t, r.Audit.Path)
	if rec.CommitScope != repoRoot {
		t.Errorf("commit_scope = %q, want %q (the matched child repo, not cwd's toplevel)",
			rec.CommitScope, repoRoot)
	}
	if rec.Verb != "repo.noop" {
		t.Errorf("verb = %q, want repo.noop", rec.Verb)
	}
	if rec.Decision != audit.DecisionAccept {
		t.Errorf("decision = %q, want accept", rec.Decision)
	}
}

// TestLoadRepoExecCommand_NoConfigStillBuildsExec proves the verb stays
// visible in --help even when neither cwd's ancestry nor its direct
// children declare a coily.yaml. The Action returns a UserError so the
// operator gets a clear recovery hint instead of "command not found".
func TestLoadRepoExecCommand_NoConfigStillBuildsExec(t *testing.T) {
	parent := t.TempDir()
	t.Setenv(repocfg.EnvOverride, "")
	pushdir(t, parent)

	r := newTestRunner(t)
	res, exec := r.loadRepoExecCommand()
	if exec == nil {
		t.Fatal("loadRepoExecCommand returned nil exec command")
	}
	if res.Ancestor != nil {
		t.Errorf("Ancestor = %v, want nil", res.Ancestor)
	}
	if len(res.Children) != 0 {
		t.Errorf("Children = %v, want empty", res.Children)
	}
	if exec.Action == nil {
		t.Fatal("stub exec command has no Action")
	}
	err := exec.Action(context.Background(), exec)
	if err == nil {
		t.Fatal("stub exec Action returned nil; expected UserError")
	}
	if k := errKind(err); k != "repo_no_config" {
		t.Errorf("err kind = %q, want repo_no_config", k)
	}
}

// TestLoadRepoExecCommand_DiscoversChildren proves the headline case
// from issue #102: `coily exec` invoked one directory above a child
// repo discovers that child and registers its commands as subcommands.
func TestLoadRepoExecCommand_DiscoversChildren(t *testing.T) {
	parent := t.TempDir()
	mkChild(t, parent, "alpha", "commands:\n  alpha-only: true\n")
	t.Setenv(repocfg.EnvOverride, "")
	pushdir(t, parent)

	r := newTestRunner(t)
	res, exec := r.loadRepoExecCommand()
	if res.Ancestor != nil {
		t.Errorf("Ancestor = %v, want nil", res.Ancestor)
	}
	if len(res.Children) != 1 {
		t.Fatalf("Children = %v, want one entry", res.Children)
	}
	names := make([]string, 0, len(exec.Commands))
	for _, c := range exec.Commands {
		names = append(names, c.Name)
	}
	if len(names) != 1 || names[0] != "alpha-only" {
		t.Errorf("exec subcommands = %v, want [alpha-only]", names)
	}
}

func mkChild(t *testing.T, parent, name, body string) {
	t.Helper()
	overlay := filepath.Join(parent, name, repocfg.LocalDirName)
	if err := os.MkdirAll(overlay, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", overlay, err)
	}
	if err := os.WriteFile(filepath.Join(overlay, repocfg.Filename), []byte(body), 0o644); err != nil {
		t.Fatalf("write coily.yaml: %v", err)
	}
}

// pushdir chdirs to dir for the duration of the test, restoring the
// original cwd on cleanup. Used to drive loadRepoExecCommand without
// reaching past its os.Getwd dependency.
func pushdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(prev); err != nil {
			t.Logf("restore cwd: %v", err)
		}
	})
}

// wrapExecRoot wraps a single cli.Command as the only top-level subcommand
// of a `coily` root carrying the same global flags the production main
// installs. Tests that exercise an `exec` subtree pass the result of
// buildExecFromChildren directly; tests that exercise a single
// auto-executing leaf pass the buildChildRepoCommand result and invoke
// it as a top-level verb so c.Root() resolves correctly inside the
// verb pipeline.
func wrapExecRoot(child *cli.Command) *cli.Command {
	return &cli.Command{
		Name: "coily",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "audit-override-dirty"},
			&cli.StringFlag{Name: "commit-scope", Value: "auto"},
		},
		Commands: []*cli.Command{child},
	}
}
