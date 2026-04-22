//go:build integration

// Package integration_test holds Layer 2 integration tests that exercise the
// coily pass-through pipeline against the live aws/gh/kubectl CLIs installed
// on the host. Run with:
//
//	make test-integration
//
// or
//
//	go test -tags integration ./test/integration/...
//
// These tests call whoami-style verbs (aws sts get-caller-identity, gh api
// user, kubectl config current-context). whoami is the safest read-only
// verb that exists on every target CLI, so using it as the smoke-test
// vehicle proves the whole pipeline works end-to-end (policy check, argv
// construction, subprocess exec, output normalization) without touching
// any mutable state.
package integration_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestWhoamiReturnsYAMLWithExpectedKeys runs `coily whoami` and verifies
// the output is valid yaml with top-level blocks for aws, kubectl, and gh.
// Does not assert on specific identities because those depend on the
// environment and will change across machines.
func TestWhoamiReturnsYAMLWithExpectedKeys(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "run", "./cmd/coily", "whoami")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("coily whoami failed: %v\noutput:\n%s", err, out)
	}

	var v map[string]any
	if err := yaml.Unmarshal(out, &v); err != nil {
		t.Fatalf("output is not yaml: %v\noutput:\n%s", err, out)
	}

	for _, key := range []string{"aws", "kubectl", "gh"} {
		block, ok := v[key]
		if !ok {
			t.Errorf("missing top-level key %q in output", key)
			continue
		}
		if block == nil {
			t.Errorf("block %q is nil", key)
		}
	}
}

// TestWhoamiBlocksContainIdentityOrError asserts each block either carries
// an identity field (proves the subprocess ran and returned real data) or
// an "error" field (proves the error path is reachable and structured).
// Tolerant of unconfigured tools, strict about shape.
func TestWhoamiBlocksContainIdentityOrError(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/coily", "whoami")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("coily whoami failed: %v", err)
	}
	var v map[string]map[string]any
	if err := yaml.Unmarshal(out, &v); err != nil {
		t.Fatalf("parse: %v\n%s", err, out)
	}

	expectedIdentity := map[string][]string{
		"aws":     {"Account", "Arn", "UserId"},
		"gh":      {"login", "id"},
		"kubectl": {"current_context", "username"}, // current_context (fallback path) OR username (auth whoami path)
	}

	for tool, identityKeys := range expectedIdentity {
		block := v[tool]
		if block == nil {
			t.Errorf("%s block missing", tool)
			continue
		}
		if _, hasErr := block["error"]; hasErr {
			// Error path is a valid shape.
			t.Logf("%s block surfaced an error (ok, tool may be unconfigured): %v", tool, block["error"])
			continue
		}
		found := false
		for _, k := range identityKeys {
			if _, ok := block[k]; ok {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s block has neither an error nor an identity key (%s). Got keys: %v",
				tool, strings.Join(identityKeys, "/"), keys(block))
		}
	}
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
