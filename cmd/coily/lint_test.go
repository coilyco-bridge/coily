package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMakefileTargets(t *testing.T) {
	dir := t.TempDir()
	mk := filepath.Join(dir, "Makefile")
	body := strings.Join([]string{
		"VERSION := dev",
		"",
		"# leading comment, no target here",
		".PHONY: build",
		"build: ## Build the binary.",
		"\tgo build .",
		"",
		"test: deps ## Run tests.",
		"\tgo test ./...",
		"",
		"undocumented:",
		"\techo skip",
		"",
	}, "\n")
	if err := os.WriteFile(mk, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := loadMakefileTargets(mk)
	if err != nil {
		t.Fatal(err)
	}
	if got["build"].description != "Build the binary." {
		t.Errorf("build desc = %q", got["build"].description)
	}
	if got["test"].description != "Run tests." {
		t.Errorf("test desc = %q", got["test"].description)
	}
	if _, ok := got["undocumented"]; ok {
		t.Errorf("undocumented should not be picked up without ## desc")
	}
}

func TestLoadCoilyYamlVerbs_EmptyCommandsIsNoop(t *testing.T) {
	dir := t.TempDir()
	overlay := filepath.Join(dir, ".coily")
	if err := os.MkdirAll(overlay, 0o755); err != nil {
		t.Fatal(err)
	}
	yamlPath := filepath.Join(overlay, "coily.yaml")
	for _, body := range []string{
		"commands: {}\n",
		"commands:\n",
		"version: 1\n",
	} {
		if err := os.WriteFile(yamlPath, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		verbs, err := loadCoilyYamlVerbs(yamlPath)
		if err != nil {
			t.Fatalf("body=%q: %v", body, err)
		}
		if len(verbs) != 0 {
			t.Errorf("body=%q: got %d verbs, want 0", body, len(verbs))
		}
	}
}

func TestLoadCoilyYamlVerbs(t *testing.T) {
	dir := t.TempDir()
	overlay := filepath.Join(dir, ".coily")
	if err := os.MkdirAll(overlay, 0o755); err != nil {
		t.Fatal(err)
	}
	yamlPath := filepath.Join(overlay, "coily.yaml")
	body := `commands:
  test:
    run: make test
    description: Run tests.
  lint:
    run: make lint
    description: Lint.
`
	if err := os.WriteFile(yamlPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	verbs, err := loadCoilyYamlVerbs(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(verbs) != 2 {
		t.Fatalf("got %d verbs, want 2", len(verbs))
	}
	if verbs[0].name != "test" || verbs[0].run != "make test" || verbs[0].description != "Run tests." {
		t.Errorf("verb[0] = %+v", verbs[0])
	}
	if verbs[1].name != "lint" || verbs[1].run != "make lint" {
		t.Errorf("verb[1] = %+v", verbs[1])
	}
}
