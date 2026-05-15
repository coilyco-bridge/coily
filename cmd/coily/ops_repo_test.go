package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/cli-guard/audit"
	"github.com/coilysiren/cli-guard/repocfg"
	"github.com/coilysiren/cli-guard/shell"
	"github.com/urfave/cli/v3"
)

// TestBuildExecFromConfigs_AggregatesCommands proves that the unified
// discovery pool yields a usable exec command tree: each unique command
// name across the pool becomes a subcommand, sorted by name.
func TestBuildExecFromConfigs_AggregatesCommands(t *testing.T) {
	parent := t.TempDir()
	mkChild(t, parent, "alpha", "commands:\n  alpha-only: true\n  shared: true\n")
	mkChild(t, parent, "beta", "commands:\n  beta-only: true\n  shared: true\n")

	cfgs, err := repocfg.DiscoverAll(parent)
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}
	r := newTestRunner(t)
	exec := r.buildExecFromConfigs(cfgs)

	want := []string{"alpha-only", "beta-only", "shared"}
	got := make([]string, 0, len(exec.Commands))
	for _, c := range exec.Commands {
		got = append(got, c.Name)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("subcommand names = %v, want %v", got, want)
	}
}

// TestPromptChildChoice_PicksByIndex covers the headline pause-for-input
// behavior: when a name is declared by multiple repos, the subcommand
// prompts on stderr with a numbered menu and reads a 1-indexed pick
// from stdin. Drives promptChildChoice directly so the test doesn't
// depend on a real git repo for the gittree gate.
func TestPromptChildChoice_PicksByIndex(t *testing.T) {
	parent := t.TempDir()
	mkChild(t, parent, "alpha", "commands:\n  shared: true\n")
	mkChild(t, parent, "beta", "commands:\n  shared: true\n")
	cfgs, err := repocfg.DiscoverAll(parent)
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}
	matches := []childMatch{
		{cfg: cfgs[0], cmd: cfgs[0].Commands[0]},
		{cfg: cfgs[1], cmd: cfgs[1].Commands[0]},
	}
	var out bytes.Buffer
	pick, err := promptChildChoice("shared", matches, strings.NewReader("2\n"), &out)
	if err != nil {
		t.Fatalf("promptChildChoice: %v", err)
	}
	wantRoot := filepath.Dir(filepath.Dir(matches[1].cfg.Path))
	gotRoot := filepath.Dir(filepath.Dir(pick.cfg.Path))
	if gotRoot != wantRoot {
		t.Errorf("picked repo root = %q, want %q", gotRoot, wantRoot)
	}
	if !strings.Contains(out.String(), "pick one") || !strings.Contains(out.String(), "choice [1-2]") {
		t.Errorf("prompt output missing expected lines:\n%s", out.String())
	}
}

// TestPromptChildChoice_RejectsOutOfRange proves the prompt errors with
// exec_prompt_invalid rather than panicking when the agent feeds back a
// number outside the menu range. The hint names the valid range so the
// next retry can succeed.
func TestPromptChildChoice_RejectsOutOfRange(t *testing.T) {
	parent := t.TempDir()
	mkChild(t, parent, "alpha", "commands:\n  shared: true\n")
	mkChild(t, parent, "beta", "commands:\n  shared: true\n")
	cfgs, err := repocfg.DiscoverAll(parent)
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}
	matches := []childMatch{
		{cfg: cfgs[0], cmd: cfgs[0].Commands[0]},
		{cfg: cfgs[1], cmd: cfgs[1].Commands[0]},
	}
	_, err = promptChildChoice("shared", matches, strings.NewReader("99\n"), io.Discard)
	if err == nil {
		t.Fatal("expected error for out-of-range pick, got nil")
	}
	if k := errKind(err); k != "exec_prompt_invalid" {
		t.Errorf("err kind = %q, want exec_prompt_invalid", k)
	}
}

// TestBuildExecFromConfigs_PromptingSubcommandPicksRepo wires the
// prompting subcommand through verb.Wrap and proves that after stdin
// feeds "2\n", the audit row's commit-scope binds to the second repo
// (the one the operator picked), not the first.
func TestBuildExecFromConfigs_PromptingSubcommandPicksRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	if _, err := exec.LookPath("true"); err != nil {
		t.Skip("true not on PATH")
	}

	repoA := initSecurityClaimRepo(t)
	repoB := initSecurityClaimRepo(t)
	for _, root := range []string{repoA, repoB} {
		cfgDir := filepath.Join(root, ".coily")
		if err := os.MkdirAll(cfgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cfgDir, "coily.yaml"),
			[]byte("commands:\n  noop: true\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		mustGitForClaim(t, root, "add", ".coily/coily.yaml")
		mustGitForClaim(t, root, "commit", "-m", "add coily.yaml")
		mustGitForClaim(t, root, "push")
	}
	cfgA, err := repocfg.Load(filepath.Join(repoA, ".coily", "coily.yaml"))
	if err != nil {
		t.Fatalf("load A: %v", err)
	}
	cfgB, err := repocfg.Load(filepath.Join(repoB, ".coily", "coily.yaml"))
	if err != nil {
		t.Fatalf("load B: %v", err)
	}
	// Order matches alphanumeric by Path. The test feeds "2" so we
	// require the "second" match to be repoB regardless of which repo
	// the OS-given tempdir ordering put first.
	configs := []*repocfg.Config{cfgA, cfgB}
	// Stable sort by Path to mirror DiscoverAll's contract.
	if cfgA.Path > cfgB.Path {
		configs = []*repocfg.Config{cfgB, cfgA}
	}

	r := newSecurityClaimRunnerWithAudit(t)
	r.Runner = &shell.Runner{
		Stdout: os.Stdout,
		Stderr: io.Discard,
		Stdin:  strings.NewReader("2\n"),
	}
	execCmd := r.buildExecFromConfigs(configs)
	root := wrapExecRoot(execCmd)
	// exec subcommand "noop" is the prompting variant.
	if err := root.Run(t.Context(), []string{"coily", "exec", "noop"}); err != nil {
		t.Fatalf("prompting run: %v", err)
	}
	rec := lastAuditRecord(t, r.Audit.Path)
	wantScope := filepath.Dir(filepath.Dir(configs[1].Path))
	if rec.CommitScope != wantScope {
		t.Errorf("commit_scope = %q, want %q (the picked repo)", rec.CommitScope, wantScope)
	}
	if rec.Verb != "repo.noop" {
		t.Errorf("verb = %q, want repo.noop", rec.Verb)
	}
	if rec.Decision != audit.DecisionAccept {
		t.Errorf("decision = %q, want accept", rec.Decision)
	}
}

// TestBuildChildRepoCommand_BindsAuditScopeToChild is the headline-case
// invariant: running `coily exec daily-social` from above agentic-os-kai
// (the issue's example) must bind the audit row's commit-scope to the
// matched repo, not cwd's git toplevel.
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
		t.Errorf("commit_scope = %q, want %q (the matched repo, not cwd's toplevel)",
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
// visible in --help even when nothing is reachable from cwd's discovery
// pool. The Action returns a UserError so the operator gets a clear
// recovery hint instead of "command not found".
func TestLoadRepoExecCommand_NoConfigStillBuildsExec(t *testing.T) {
	parent := t.TempDir()
	t.Setenv(repocfg.EnvOverride, "")
	pushdir(t, parent)

	r := newTestRunner(t)
	res, exec := r.loadRepoExecCommand()
	if exec == nil {
		t.Fatal("loadRepoExecCommand returned nil exec command")
	}
	if len(res.Configs) != 0 {
		t.Errorf("Configs = %v, want empty", res.Configs)
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
// from issue #102 still works under the unified discovery pool: `coily
// exec` invoked one directory above a child repo discovers that child
// and registers its commands as subcommands.
func TestLoadRepoExecCommand_DiscoversChildren(t *testing.T) {
	parent := t.TempDir()
	mkChild(t, parent, "alpha", "commands:\n  alpha-only: true\n")
	t.Setenv(repocfg.EnvOverride, "")
	pushdir(t, parent)

	r := newTestRunner(t)
	res, exec := r.loadRepoExecCommand()
	if len(res.Configs) != 1 {
		t.Fatalf("Configs = %v, want one entry", res.Configs)
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

// wrapExecRoot wraps a single cli.Command as the only top-level
// subcommand of a `coily` root carrying the same global flags the
// production main installs.
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
