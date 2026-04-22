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
