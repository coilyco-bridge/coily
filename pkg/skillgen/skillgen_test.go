package skillgen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/skillgen"
)

// writeManifest drops a tiny commands/<bin>.yaml in a temp configs dir.
// Used by tests that exercise the manifest-reading code.
func writeManifest(t *testing.T, dir, binary, body string) {
	t.Helper()
	cmds := filepath.Join(dir, "commands")
	if err := os.MkdirAll(cmds, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmds, binary+".yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestGenerate_WritesSkillAndReference(t *testing.T) {
	cfgs := t.TempDir()
	out := t.TempDir()
	writeManifest(t, cfgs, "aws", `
binary: aws
bin_version: aws-cli/2.0.0
commands:
  - path: [sts]
    help: Security Token Service
    children: [get-caller-identity]
  - path: [sts, get-caller-identity]
    help: Returns details about the IAM user or role whose credentials were used.
    flags:
      - name: --profile
      - name: --output
`)
	err := skillgen.Generate(skillgen.Options{
		CommandsDir: filepath.Join(cfgs, "commands"),
		OutDir:      out,
		Verbs: []skillgen.Verb{
			{Name: "lockdown", Usage: "Install perms", Flags: []string{"--apply"}, Example: "coily lockdown --apply"},
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	skill := readFile(t, filepath.Join(out, "SKILL.md"))
	ref := readFile(t, filepath.Join(out, "reference", "aws.md"))

	for _, want := range []string{
		"name: coily",
		"## Coily-native verbs",
		"coily lockdown",
		"Pass-through tools",
		"coily aws",
	} {
		if !strings.Contains(skill, want) {
			t.Errorf("SKILL.md missing %q", want)
		}
	}
	for _, want := range []string{
		"# coily aws - full reference",
		"sts get-caller-identity",
		"--profile",
		"Returns details about the IAM user",
	} {
		if !strings.Contains(ref, want) {
			t.Errorf("reference missing %q", want)
		}
	}
}

// TestGenerate_HoistsGlobalFlags verifies that flags appearing on every
// leaf collapse into a single "Global flags" section and stop appearing
// on each per-verb subsection. Without this, aws.md repeats ~17 globals
// across 323 verbs (~5,500 redundant lines).
func TestGenerate_HoistsGlobalFlags(t *testing.T) {
	cfgs := t.TempDir()
	out := t.TempDir()
	// Three leaves, each with three shared global flags
	// (--profile / --region / --output) plus one verb-specific flag.
	writeManifest(t, cfgs, "aws", `
binary: aws
bin_version: aws-cli/2.0.0
commands:
  - path: [sts]
    help: Security Token Service
    children: [get-caller-identity, get-session-token, assume-role]
  - path: [sts, get-caller-identity]
    help: Returns details about the IAM user.
    flags:
      - name: --profile
      - name: --region
      - name: --output
  - path: [sts, get-session-token]
    help: Returns a temporary session token.
    flags:
      - name: --duration-seconds
      - name: --profile
      - name: --region
      - name: --output
  - path: [sts, assume-role]
    help: Returns temporary credentials.
    flags:
      - name: --role-arn
      - name: --profile
      - name: --region
      - name: --output
`)
	err := skillgen.Generate(skillgen.Options{
		CommandsDir: filepath.Join(cfgs, "commands"),
		OutDir:      out,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	ref := readFile(t, filepath.Join(out, "reference", "aws.md"))

	if !strings.Contains(ref, "## Global flags") {
		t.Error("reference missing Global flags section")
	}
	// Globals appear exactly once - in the Global flags block, not under
	// any per-verb section.
	if got := strings.Count(ref, "- `--profile`"); got != 1 {
		t.Errorf("--profile appeared %d times, want 1 (in Global flags only)", got)
	}
	if got := strings.Count(ref, "- `--region`"); got != 1 {
		t.Errorf("--region appeared %d times, want 1", got)
	}
	// Verb-specific flags are still listed under their verbs.
	if !strings.Contains(ref, "- `--duration-seconds`") {
		t.Error("verb-specific --duration-seconds missing")
	}
	if !strings.Contains(ref, "- `--role-arn`") {
		t.Error("verb-specific --role-arn missing")
	}
}

// TestGenerate_GlobalFlagsThreshold confirms the hoist is suppressed
// when there are too few leaves for the dedup to be worth the extra
// navigation. A two-leaf manifest renders flags inline.
func TestGenerate_GlobalFlagsThreshold(t *testing.T) {
	cfgs := t.TempDir()
	out := t.TempDir()
	writeManifest(t, cfgs, "tiny", `
binary: tiny
commands:
  - path: [foo]
    flags:
      - name: --shared
      - name: --foo-only
  - path: [bar]
    flags:
      - name: --shared
      - name: --bar-only
`)
	if err := skillgen.Generate(skillgen.Options{
		CommandsDir: filepath.Join(cfgs, "commands"),
		OutDir:      out,
	}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	ref := readFile(t, filepath.Join(out, "reference", "tiny.md"))
	if strings.Contains(ref, "## Global flags") {
		t.Error("Global flags section should not appear with only 2 leaves")
	}
	if got := strings.Count(ref, "- `--shared`"); got != 2 {
		t.Errorf("--shared inline count = %d, want 2", got)
	}
}

func TestGenerate_NoVerbsNoManifests(t *testing.T) {
	cfgs := t.TempDir()
	_ = os.MkdirAll(filepath.Join(cfgs, "commands"), 0o755)
	out := t.TempDir()
	err := skillgen.Generate(skillgen.Options{
		CommandsDir: filepath.Join(cfgs, "commands"),
		OutDir:      out,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	skill := readFile(t, filepath.Join(out, "SKILL.md"))
	// Even with no input, SKILL.md should be produced with the frontmatter +
	// basic structure so Claude Code has something to load.
	if !strings.Contains(skill, "name: coily") {
		t.Errorf("SKILL.md missing frontmatter")
	}
	if strings.Contains(skill, "Pass-through tools") {
		t.Errorf("SKILL.md should omit Pass-through section when no manifests")
	}
}

func TestGenerate_SkipsEmptyBinary(t *testing.T) {
	cfgs := t.TempDir()
	out := t.TempDir()
	writeManifest(t, cfgs, "valid", "binary: valid\ncommands: []\n")
	writeManifest(t, cfgs, "empty", "commands: []\n") // no binary
	err := skillgen.Generate(skillgen.Options{
		CommandsDir: filepath.Join(cfgs, "commands"),
		OutDir:      out,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "reference", "valid.md")); err != nil {
		t.Errorf("valid.md was not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "reference", "empty.md")); err == nil {
		t.Errorf("empty binary should have been skipped")
	}
}

func TestGenerate_MissingCommandsDir(t *testing.T) {
	out := t.TempDir()
	err := skillgen.Generate(skillgen.Options{
		CommandsDir: "/does/not/exist/coily-test",
		OutDir:      out,
	})
	if err == nil {
		t.Error("expected error for missing commands dir")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
