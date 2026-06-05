package main

// Tests in this file walk SECURITY.md's load-bearing claims and assert each
// against runtime artifacts. The point is to catch prose-runtime drift: when
// a feature is trimmed and the prose forgets to follow, this test fails
// before the boundary description ships out of date.
//
// When SECURITY.md gains a testable claim, add a TestSecurityClaim_*
// function here. When a SECURITY.md claim gets deleted, delete the
// corresponding test. The two should move together.
//
// What "load-bearing" means here: a claim about runtime behavior whose
// failure would weaken the boundary or mislead an operator. Out of scope:
// rationale paragraphs, threat-model background, references to deny rules
// that live in another file.

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/audit"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/gittree"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/lockdown"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/policy"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/repocfg"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/shell"
	"github.com/urfave/cli/v3"
)

// newSecurityClaimRunner builds a Runner sufficient for command-tree
// walking. Loads the layered config (defaults + any host overlays) so verb
// builders that dereference r.Cfg do not panic. A shell.Runner is supplied
// because dispatchCommand wires it into dispatch.New at tree-build time, and
// dispatch.New refuses a nil Runner. Audit stays nil; tests in this
// file do not invoke Actions.
func newSecurityClaimRunner(t *testing.T) *Runner {
	t.Helper()
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	return &Runner{
		Cfg:    cfg,
		Runner: &shell.Runner{Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin},
	}
}

// TestSecurityClaim_PolicyRejectsAllShellMetacharacters covers the
// SECURITY.md claim that cli-guard/policy rejects shell metacharacters before they
// reach a subprocess. Walks every byte in the documented ShellMeta set so a
// loosened matcher fails here even if the matcher's own unit test missed it.
func TestSecurityClaim_PolicyRejectsAllShellMetacharacters(t *testing.T) {
	if policy.ShellMeta == "" {
		t.Fatal("policy.ShellMeta is empty; the metachar gate is wide open")
	}
	for _, b := range []byte(policy.ShellMeta) {
		probe := "x" + string(b) + "y"
		err := policy.ValidateArg("test", probe)
		if err == nil {
			t.Errorf("ValidateArg(%q) returned nil; ShellMeta byte %q (0x%02x) must be rejected",
				probe, b, b)
		}
		if !errors.Is(err, policy.ErrShellMeta) {
			t.Errorf("ValidateArg(%q) error = %v; want errors.Is(_, ErrShellMeta)", probe, err)
		}
	}
}

// TestSecurityClaim_NoEscapeHatchVerbs covers SECURITY.md's "No coily shell /
// coily run escape hatch, ever. No coily ops kubectl exec pass-through."
//
// Walks the registered command tree built by the production Runner and fails
// if any forbidden name lands as a top-level verb. (Kubectl is a passthrough
// that does not register subcommands in the tree; the deny list at
// cli-guard/lockdown/defaults.yaml covers `kubectl exec` separately and is
// asserted by TestSecurityClaim_LockdownDeniesKubectlExec.)
func TestSecurityClaim_NoEscapeHatchVerbs(t *testing.T) {
	r := newSecurityClaimRunner(t)
	cmds := r.builtInCommands()

	// "shell" and "run" must not appear as top-level coily verbs. They
	// would name an unrestricted-execution surface that defeats the
	// boundary.
	forbiddenTopLevel := map[string]bool{
		"shell": true,
		"run":   true,
		"exec":  true,
	}
	for _, c := range cmds {
		if forbiddenTopLevel[c.Name] {
			t.Errorf("forbidden top-level verb registered: coily %q (SECURITY.md: no escape hatch)", c.Name)
		}
	}
}

// TestSecurityClaim_NoConfirmationTokenVerb covers the SECURITY.md history
// note that the HMAC-token design was removed. A `coily auth issue` verb
// returning a fresh token would re-introduce the same self-authorization
// pattern that motivated the removal.
func TestSecurityClaim_NoConfirmationTokenVerb(t *testing.T) {
	r := newSecurityClaimRunner(t)
	for _, c := range r.builtInCommands() {
		if c.Name == "auth" {
			t.Errorf("coily auth verb registered; SECURITY.md says token ritual added no security")
			for _, sub := range c.Commands {
				t.Logf("  subcommand: %s", sub.Name)
			}
		}
	}
}

// TestSecurityClaim_VerbWrapIsTheChokepoint covers the architectural claim
// that every cli.Action goes through verb.Wrap. Today this is a convention
// rather than a structural enforcement (the test reads cli command Action
// pointers and counts how many distinct functions appear), so this test is
// best-effort: if it ever breaks, the fix is a code change to make every
// Action come from verb.Wrap, not a relaxation of the assertion.
//
// Implementation note: cli.ActionFunc values cannot be reliably compared by
// pointer because of closures, so we cannot assert "all Actions are the same
// function." Instead we walk the tree and verify that every leaf with an
// Action has a corresponding ArgsFunc-shaped wrap somewhere up the chain.
// This catches the most common bypass mode: a hand-built closure inlined at
// the verb registration site.
//
// For now, mark as t.Skip with a TODO so the test file documents the gap
// explicitly rather than silently lacking the check. When verb.Wrap exposes
// a marker (e.g. a wrapped *Spec attached to the command) this test can do
// real work.
func TestSecurityClaim_VerbWrapIsTheChokepoint(t *testing.T) {
	t.Skip("TODO: verb.Wrap does not yet stamp commands with a marker. " +
		"Until then, this test exists as a placeholder so the gap is visible.")
}

// TestSecurityClaim_NoDevModeBypassInProdBuilds is a build-tag check.
// SECURITY.md says production builds use -tags prod which compiles out
// any dev-mode conveniences. This test only runs the smoke path in unit
// tests; full coverage is the responsibility of the prod-tag build itself.
//
// Documented here as a placeholder so the prose claim is visible to anyone
// auditing what the test file covers.
func TestSecurityClaim_NoDevModeBypassInProdBuilds(t *testing.T) {
	t.Skip("Build-tag separation is enforced by the build system, not by a runtime test.")
}

// TestSecurityClaim_UserBinaryGateUnconditional covers the SECURITY.md
// claim that the user-level coily-binary-gate hook fires for every Bash
// PreToolUse event - including cron-spawned local-agent sessions.
//
// Two concrete checks back the claim:
//
//  1. The settings.json entry written by EnsureUserHook registers a
//     PreToolUse Bash matcher with no transcript_path / time-of-day /
//     other conditional skip. The companion UserPromptSubmit cron-bypass
//     fix (2026-05-08) must not have leaked into this entry.
//  2. The hook script body itself does not early-return on a
//     transcript_path or local-agent-mode marker. If a future change
//     adds such a skip, this test catches it.
//
// What this does NOT cover: whether Claude Code Desktop / cron-spawned
// agent sessions actually invoke the user-level hooks at all. That is
// runtime behavior of an external harness; the operational verification
// step is documented in issue #66 and lives outside the unit test.
// Issue #66.
func TestSecurityClaim_UserBinaryGateUnconditional(t *testing.T) {
	home := t.TempDir()
	hookPath, _, err := lockdown.EnsureUserHook(home, coilyLockdownDriver())
	if err != nil {
		t.Fatalf("EnsureUserHook: %v", err)
	}

	// (1) settings.json entry: matcher = "Bash" with no conditional fields.
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("parse settings: %v", err)
	}
	hooks, _ := root["hooks"].(map[string]any)
	preToolUse, _ := hooks["PreToolUse"].([]any)
	var ourEntry map[string]any
	for _, e := range preToolUse {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		inner, _ := m["hooks"].([]any)
		for _, h := range inner {
			hm, _ := h.(map[string]any)
			if marker, _ := hm["_coily"].(string); marker == coilyLockdownDriver().UserHookMarkerKey {
				ourEntry = m
				break
			}
		}
	}
	if ourEntry == nil {
		t.Fatalf("user-level coily-binary-gate hook not registered; got settings: %s", string(raw))
	}
	if matcher, _ := ourEntry["matcher"].(string); matcher != "Bash" {
		t.Errorf("hook matcher = %q, want %q (cron sessions need the same Bash gate)", matcher, "Bash")
	}
	// Any field beyond matcher/hooks would be a conditional that could
	// silently exclude cron-spawned sessions. The schema today is
	// {matcher, hooks} only; surface unexpected keys here so a future
	// schema-extension that adds (e.g.) a "when" clause has to be reviewed.
	for k := range ourEntry {
		if k != "matcher" && k != "hooks" {
			t.Errorf("user-level hook entry carries unexpected field %q; conditional fields could exclude cron sessions", k)
		}
	}

	// (2) Hook script body: no transcript_path / local-agent skip.
	body, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	for _, marker := range []string{
		"transcript_path",
		"local-agent-mode-sessions",
		"late-ok",
	} {
		if strings.Contains(string(body), marker) {
			t.Errorf("hook script contains %q; the binary gate must fire unconditionally for cron sessions", marker)
		}
	}
}

// TestSecurityClaim_UserBinaryGateBlocksDevCoilyForCronStdin pins the
// runtime behavior of the user-level hook against synthetic stdin shaped
// like a cron-spawned session's PreToolUse event. The harness passes the
// transcript path in `transcript_path`; the hook ignores that field and
// still rejects a dev coily binary.
//
// Companion to TestSecurityClaim_UserBinaryGateUnconditional: the prior
// test asserts the hook is registered without a skip, this one asserts
// the hook actually rejects a forbidden invocation when the input shape
// looks cron-spawned. Issue #66.
func TestSecurityClaim_UserBinaryGateBlocksDevCoilyForCronStdin(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("/bin/sh not available")
	}
	home := t.TempDir()
	hookPath, _, err := lockdown.EnsureUserHook(home, coilyLockdownDriver())
	if err != nil {
		t.Fatalf("EnsureUserHook: %v", err)
	}

	// Cron-style PreToolUse stdin: includes transcript_path under the
	// cron-spawned local-agent-mode-sessions tree. The hook must ignore
	// that field and reject the dev coily.
	stdin := `{"transcript_path":"/local-agent-mode-sessions/abc123/t.jsonl",` +
		`"tool_input":{"command":"/Users/someone/go/bin/coily version"}}`
	cmd := exec.Command("sh", hookPath) //nolint:gosec // hookPath is generated under t.TempDir
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("hook accepted dev coily for cron-style stdin; output: %s", out)
	}
	if !strings.Contains(string(out), "lockdown: blocked") {
		t.Errorf("hook output does not name lockdown block; got: %s", out)
	}
}

// TestSecurityClaim_InlineAuthoringBlocked covers the SECURITY.md claim that
// the lockdown hook denies inline shell function definitions and eval -
// inline program-authoring that runs with no on-disk artifact to inspect.
// The leading-token model is blind to it (a function name and "eval" look
// benign as leading tokens), so the check runs on the raw command before
// the segment split, in renderHookHeader, covering both the per-repo and the
// unconditional user-level hook. cli-guard#51.
//
// Drives the rendered user hook end-to-end: funcdef + eval -> blocked; a
// plain loop and eval-as-argument -> allowed. Companion runtime pin for the
// hook-generator change vendored from cli-guard.
func TestSecurityClaim_InlineAuthoringBlocked(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("/bin/sh not available")
	}
	home := t.TempDir()
	hookPath, _, err := lockdown.EnsureUserHook(home, coilyLockdownDriver())
	if err != nil {
		t.Fatalf("EnsureUserHook: %v", err)
	}
	cases := []struct {
		name      string
		command   string
		wantBlock bool
	}{
		{"inline funcdef + call", `f() { ls; }; f`, true},
		{"function-keyword def", `function f { ls; }`, true},
		{"eval of var", `code=ls; eval "$code"`, true},
		{"plain loop allowed", `for d in a b; do echo $d; done`, false},
		{"eval as argument allowed", `grep eval notes.txt`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmdJSON, _ := json.Marshal(tc.command)
			stdin := `{"tool_input":{"command":` + string(cmdJSON) + `}}`
			cmd := exec.Command("sh", hookPath) //nolint:gosec // hookPath is generated under t.TempDir
			cmd.Stdin = strings.NewReader(stdin)
			out, runErr := cmd.CombinedOutput()
			blocked := runErr != nil
			if blocked != tc.wantBlock {
				t.Fatalf("blocked=%v want %v; output: %s", blocked, tc.wantBlock, out)
			}
			if tc.wantBlock && !strings.Contains(string(out), "lockdown: blocked") {
				t.Errorf("block did not name lockdown; got: %s", out)
			}
		})
	}
}

// TestSecurityClaim_repo_verbs_require_clean_tree covers SECURITY.md's
// claim that .coily/coily.yaml repo verbs refuse to run when the audit
// row could not be reconstructed from git history. The gate exists so
// the verb argv (declared in coily.yaml) is always recoverable at the
// HEAD commit, closing the off-host shadow that local edits to the
// declaring file would otherwise create.
//
// Drives the gate end-to-end: build a tiny repo, drop a coily.yaml with a
// no-op verb, run the verb under buildRepoCommand. Asserts:
//
//  1. Clean repo: the verb runs and the audit row carries no override.
//  2. Dirty repo, only non-coily.yaml files dirty: verb runs without
//     override; audit row captures porcelain status without
//     audit_override=true (coilysiren/coily#211).
//  3. Dirty coily.yaml without --audit-override-dirty: refusal with
//     PolicyDenied.
//  4. Dirty coily.yaml with --audit-override-dirty: the verb runs and the
//     audit row is tagged audit_override=true with the porcelain status
//     captured.
func TestSecurityClaim_repo_verbs_require_clean_tree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	if _, err := exec.LookPath("true"); err != nil {
		t.Skip("/usr/bin/true not on PATH")
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
	// Commit the coily.yaml so the tree is clean.
	mustGitForClaim(t, repoRoot, "add", ".coily/coily.yaml")
	mustGitForClaim(t, repoRoot, "commit", "-m", "add coily.yaml")
	mustGitForClaim(t, repoRoot, "push")
	if st, err := gittree.CheckClean(repoRoot); err != nil || !st.Clean {
		t.Fatalf("baseline clean check failed: state=%+v err=%v", st, err)
	}

	cfg, err := repocfg.Load(cfgPath)
	if err != nil {
		t.Fatalf("repocfg.Load: %v", err)
	}
	if len(cfg.Commands) != 1 {
		t.Fatalf("want 1 command in coily.yaml, got %d", len(cfg.Commands))
	}

	// (1) Clean tree: verb runs, audit row not tagged.
	r := newSecurityClaimRunnerWithAudit(t)
	cmd := r.buildChildRepoCommand(cfg, cfg.Commands[0])
	root := wrapInRoot(cmd)
	if err := root.Run(t.Context(), []string{"coily", "noop"}); err != nil {
		t.Fatalf("clean-tree run: %v", err)
	}
	rec := lastAuditRecord(t, r.Audit.Path)
	if rec.AuditOverride {
		t.Errorf("clean run tagged audit_override=true unexpectedly")
	}
	if rec.WorkingTreeStatus != "" {
		t.Errorf("clean run captured WorkingTreeStatus=%q", rec.WorkingTreeStatus)
	}

	// (2) Dirty tree limited to non-coily.yaml files: no override needed,
	// verb runs, audit row captures status but is not tagged as override.
	// Closes coilysiren/coily#211: dirt outside the declaring file does
	// not break audit-row reconstruction.
	if err := os.WriteFile(filepath.Join(repoRoot, "dirt.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r = newSecurityClaimRunnerWithAudit(t)
	cmd = r.buildChildRepoCommand(cfg, cfg.Commands[0])
	root = wrapInRoot(cmd)
	if err := root.Run(t.Context(), []string{"coily", "noop"}); err != nil {
		t.Fatalf("dirty-outside-coily.yaml run without override: %v", err)
	}
	rec = lastAuditRecord(t, r.Audit.Path)
	if rec.AuditOverride {
		t.Errorf("non-coily.yaml dirt tagged audit_override=true; rec=%+v", rec)
	}
	if !strings.Contains(rec.WorkingTreeStatus, "dirt.txt") {
		t.Errorf("dirty-outside-coily.yaml run did not capture status mentioning dirt.txt; got %q", rec.WorkingTreeStatus)
	}

	// (3) Dirty coily.yaml without override: refusal.
	if err := os.WriteFile(cfgPath, []byte("commands:\n  noop: true\n# edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r = newSecurityClaimRunnerWithAudit(t)
	cmd = r.buildChildRepoCommand(cfg, cfg.Commands[0])
	root = wrapInRoot(cmd)
	err = root.Run(t.Context(), []string{"coily", "noop"})
	if err == nil {
		t.Fatal("dirty coily.yaml run without override returned nil; expected PolicyDenied")
	}
	if k := errKind(err); k != "repo_verb_dirty" {
		t.Errorf("dirty coily.yaml refusal kind = %q, want repo_verb_dirty (err=%v)", k, err)
	}

	// (4) Dirty coily.yaml with override: runs, audit row tagged.
	r = newSecurityClaimRunnerWithAudit(t)
	cmd = r.buildChildRepoCommand(cfg, cfg.Commands[0])
	root = wrapInRoot(cmd)
	if err := root.Run(t.Context(), []string{"coily", "--audit-override-dirty", "noop"}); err != nil {
		t.Fatalf("dirty coily.yaml run with override: %v", err)
	}
	rec = lastAuditRecord(t, r.Audit.Path)
	if !rec.AuditOverride {
		t.Errorf("override run did not tag audit_override=true; rec=%+v", rec)
	}
	if !strings.Contains(rec.WorkingTreeStatus, "coily.yaml") {
		t.Errorf("override run did not capture porcelain status mentioning coily.yaml; got %q", rec.WorkingTreeStatus)
	}
}

func errKind(err error) string {
	type kinder interface{ Kind() string }
	var k kinder
	if errors.As(err, &k) {
		return k.Kind()
	}
	return ""
}

func wrapInRoot(child *cli.Command) *cli.Command {
	return &cli.Command{
		Name: "coily",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "audit-override-dirty"},
		},
		Commands: []*cli.Command{child},
	}
}

func newSecurityClaimRunnerWithAudit(t *testing.T) *Runner {
	t.Helper()
	dir := t.TempDir()
	aw := audit.NewWriter(filepath.Join(dir, "audit.jsonl"))
	t.Cleanup(func() { _ = aw.Close() })
	return &Runner{
		Cfg:    &Config{},
		Runner: &shell.Runner{Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin},
		Audit:  aw,
	}
}

func initSecurityClaimRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	workdir := filepath.Join(root, "work")
	upstream := filepath.Join(root, "upstream.git")
	if err := os.Mkdir(workdir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustGitForClaim(t, root, "init", "--bare", "--initial-branch=main", upstream)
	mustGitForClaim(t, workdir, "init", "--initial-branch=main")
	mustGitForClaim(t, workdir, "config", "user.email", "test@example.com")
	mustGitForClaim(t, workdir, "config", "user.name", "test")
	mustGitForClaim(t, workdir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(workdir, "README"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGitForClaim(t, workdir, "add", "README")
	mustGitForClaim(t, workdir, "commit", "-m", "init")
	mustGitForClaim(t, workdir, "remote", "add", "origin", upstream)
	mustGitForClaim(t, workdir, "push", "-u", "origin", "main")
	return workdir
}

func mustGitForClaim(t *testing.T, cwd string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", cwd}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func lastAuditRecord(t *testing.T, path string) audit.Record {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open audit log: %v", err)
	}
	defer f.Close()
	recs, err := audit.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(recs) == 0 {
		t.Fatalf("audit log empty")
	}
	return recs[len(recs)-1]
}

// recordedSubcommands is a debug helper used by t.Logf in failures so the
// failing test message includes what it actually saw.
//
//nolint:unused // kept so future TestSecurityClaim_* additions can use it
func recordedSubcommands(c *cli.Command) []string {
	out := []string{}
	for _, sub := range c.Commands {
		out = append(out, sub.Name)
		for _, leaf := range sub.Commands {
			out = append(out, sub.Name+" "+leaf.Name)
		}
	}
	return out
}

// TestSecurityClaim_LockdownDeniesBareKubectlAndAwsAndGh verifies the deny
// list shipped in cli-guard/lockdown/defaults.yaml covers bare invocation of
// kubectl, aws, and gh - the three privileged-op binaries that route
// through coily ops. The previous design enumerated read-verb allows and
// write-verb denies separately because Claude Code's Bash(prefix:*) syntax
// cannot pattern-match `aws * describe-*`; the current design inverts the
// allowlist and denies the bare binaries entirely so every call lands in
// the audit log. kubectl exec, kubectl run, and the rest are covered
// transitively by Bash(kubectl:*).
//
// Note: as of coilysiren/coily#183 + coilysiren/coily#185, applyHookHandoffTrim
// removes these entries from the *rendered* per-repo settings.json so the
// coily PreToolUse hook becomes the primary gate (and surfaces the
// recovery hint on CLI, where the built-in deny matcher would otherwise
// clobber it). This test still asserts the canonical defaults shipped by
// cli-guard contain the denies; the post-trim render shape is asserted by
// TestApplyHookHandoffTrim_* in ops_lockdown_hookhandoff_test.go.
func TestSecurityClaim_LockdownDeniesBareKubectlAndAwsAndGh(t *testing.T) {
	// LoadDefaults parses cli-guard/lockdown/defaults.yaml (embedded). Asserting
	// against the parsed struct (rather than substring-matching the raw
	// file) means a typo in the file blows up here as a parse error rather
	// than as a silent miss.
	d, err := lockdown.LoadDefaults()
	if err != nil {
		t.Fatalf("lockdown.LoadDefaults: %v", err)
	}
	wantDenies := []string{
		"Bash(kubectl:*)",
		"Bash(aws:*)",
		"Bash(gh:*)",
		"Bash(flyctl:*)",
	}
	denySet := map[string]bool{}
	for _, deny := range d.Deny {
		denySet[deny] = true
	}
	for _, want := range wantDenies {
		if !denySet[want] {
			t.Errorf("lockdown defaults missing %q; the inversion routes every call through coily ops", want)
		}
	}

	// Belt-and-suspenders: assert no enumerated read-verb allow leaked back
	// in. The whole point of the inversion is "no aws / kubectl / gh
	// allows," and a strayed `Bash(aws sts get-caller-identity:*)` would
	// silently re-open the ergonomics shortcut without re-opening the
	// design conversation.
	for _, allow := range d.Allow {
		for _, prefix := range []string{"Bash(aws ", "Bash(kubectl ", "Bash(gh ", "Bash(flyctl "} {
			if strings.HasPrefix(allow, prefix) {
				t.Errorf("lockdown defaults allow %q; the inversion forbids enumerated %s reads", allow, prefix)
			}
		}
	}
}

// TestSecurityClaim_AWSReadOnlyGateDeniesSensitiveReads covers SECURITY.md's
// claim that read-only `coily ops aws` invocations are denied pre-send when
// they touch a sensitive resource pattern, and that the denial still lands
// an audit row (the trail survives the boundary). This is the #54 fix: the
// audit row used to be the only thing a read-only verb produced, documenting
// the leak without stopping it; now the read is denied before the aws CLI
// runs and the row records the denial.
//
// Two halves back the claim:
//
//  1. Wiring: the production `ops aws` passthrough registry entry carries the
//     read-only gate builder. Without this, the gate is dead code and the
//     boundary description overstates what runs.
//  2. Behavior: the gate denies a sensitive read, allows a benign one, and
//     ignores write verbs (the destructive layer is out of scope for #54).
//     A denied read lands an `ops.aws.read.denied` reject row.
func TestSecurityClaim_AWSReadOnlyGateDeniesSensitiveReads(t *testing.T) {
	// (1) Wiring: find the aws entry in the production ops registry and
	// confirm it carries the read-only gate builder.
	var awsEntry *ptEntry
	for i := range ptOps {
		if ptOps[i].Bin == "aws" {
			awsEntry = &ptOps[i]
			break
		}
	}
	if awsEntry == nil {
		t.Fatal("no aws entry in ptOps; the ops aws passthrough is gone")
	}
	if awsEntry.PreflightGateBuilder == nil {
		t.Fatal("ops aws entry has no PreflightGateBuilder; the read-only gate is not wired (coilyco-bridge/coily#54)")
	}

	// (2) Behavior: drive the gate built from a real Runner with a temp
	// audit log.
	r, logPath := newAWSGateRunner(t, AWS{})
	gate := awsEntry.PreflightGateBuilder(r)

	// Sensitive read -> hard deny.
	if err := gate([]string{"s3", "ls", "s3://prod-secrets"}); err == nil {
		t.Error("gate allowed `s3 ls` on a secrets bucket; want a pre-send deny")
	}
	// Benign read -> pass.
	if err := gate([]string{"ec2", "describe-instances"}); err != nil {
		t.Errorf("gate denied a benign read: %v", err)
	}
	// Write verb -> pass (out of scope; destructive layer is gated elsewhere).
	if err := gate([]string{"s3", "rb", "s3://prod-secrets"}); err != nil {
		t.Errorf("gate denied a write verb (out of scope for #54): %v", err)
	}

	// The denied read must have landed a reject row so the trail survives.
	rows := readAuditRows(t, logPath)
	var denied *audit.Record
	for i := range rows {
		if rows[i].Verb == "ops.aws.read.denied" {
			denied = &rows[i]
			break
		}
	}
	if denied == nil {
		t.Fatal("no ops.aws.read.denied audit row landed; the boundary silenced the trail")
	}
	if denied.Decision != audit.DecisionReject {
		t.Errorf("denied row decision = %q, want %q", denied.Decision, audit.DecisionReject)
	}
}
