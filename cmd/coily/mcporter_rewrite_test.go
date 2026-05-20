package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestRewriteMcporterArgsFile mirrors TestRewriteJQFile. The mcporter
// passthrough's `--args <json>` flag always carries `{` / `}`, so a raw
// inline payload trips the shell-metachar gate. --args-file <path> is the
// coily-side shorthand that expands into --args <content> after the gate
// has already validated the (clean) filesystem path.
func TestRewriteMcporterArgsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "args.json")
	body := `{"environment_slug":"prod","query_spec":{"calculations":[{"op":"COUNT"}]}}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	want := body[:len(body)-1] // TrimRight \n

	// Two-token form: --args-file <path>
	got := rewriteMcporterArgsFile([]string{"call", "honeycomb.run_query", "--args-file", path})
	expect := []string{"call", "honeycomb.run_query", "--args", want}
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("--args-file <path>: got %v, want %v", got, expect)
	}

	// Inline form: --args-file=<path>
	got = rewriteMcporterArgsFile([]string{"call", "honeycomb.run_query", "--args-file=" + path})
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("--args-file=<path>: got %v, want %v", got, expect)
	}

	// Missing file: fall through unchanged so mcporter reports the error itself.
	in := []string{"call", "honeycomb.run_query", "--args-file", filepath.Join(dir, "nope.json")}
	if got := rewriteMcporterArgsFile(in); !reflect.DeepEqual(got, in) {
		t.Errorf("missing file should fall through; got %v, want %v", got, in)
	}

	// No following token: fall through unchanged.
	in = []string{"call", "honeycomb.run_query", "--args-file"}
	if got := rewriteMcporterArgsFile(in); !reflect.DeepEqual(got, in) {
		t.Errorf("--args-file with no value should fall through; got %v", got)
	}

	// No --args-file at all: identity.
	in = []string{"call", "honeycomb.run_query", "--args", `{"x":1}`}
	if got := rewriteMcporterArgsFile(in); !reflect.DeepEqual(got, in) {
		t.Errorf("argv without --args-file should be returned unchanged; got %v", got)
	}
}
