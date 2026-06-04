package main

import (
	"strings"
	"testing"
)

// TestMergeForgejoLabels covers the repo-then-org fold: repo labels load
// first and an org label of the same name must not overwrite the repo
// one, while distinct org labels join the table. Bad JSON is a no-op.
func TestMergeForgejoLabels(t *testing.T) {
	m := map[string]int{}
	mergeForgejoLabels(m, []byte(`[{"id":1,"name":"P0"},{"id":2,"name":"bug"}]`))
	// org set: P0 collides (must keep repo id 1), icebox is new.
	mergeForgejoLabels(m, []byte(`[{"id":99,"name":"P0"},{"id":3,"name":"icebox"}]`))

	want := map[string]int{"P0": 1, "bug": 2, "icebox": 3}
	if len(m) != len(want) {
		t.Fatalf("size = %d, want %d (%v)", len(m), len(want), m)
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("m[%q] = %d, want %d", k, m[k], v)
		}
	}

	before := len(m)
	mergeForgejoLabels(m, []byte(`not json`))
	if len(m) != before {
		t.Errorf("bad JSON mutated the table: %v", m)
	}
}

func TestMatchForgejoLabelIDs(t *testing.T) {
	table := map[string]int{"P0": 1, "P2": 2, "icebox": 3}

	t.Run("all resolve in request order", func(t *testing.T) {
		ids, err := matchForgejoLabelIDs(table, []string{"P2", "P0"}, "px")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ids) != 2 || ids[0] != 2 || ids[1] != 1 {
			t.Errorf("ids = %v, want [2 1]", ids)
		}
	})

	t.Run("missing names surface with availability", func(t *testing.T) {
		_, err := matchForgejoLabelIDs(table, []string{"P0", "P9", "nope"}, "px")
		if err == nil {
			t.Fatal("expected an error for unresolved labels")
		}
		msg := err.Error()
		for _, want := range []string{"P9", "nope", "not found", "icebox"} {
			if !strings.Contains(msg, want) {
				t.Errorf("error missing %q: %v", want, msg)
			}
		}
		// P0 resolved, so it must not appear in the "not found: ..." segment
		// (it legitimately shows up later in "available: ...").
		notFound := msg
		if i := strings.Index(msg, "(available:"); i >= 0 {
			notFound = msg[:i]
		}
		if strings.Contains(notFound, "P0") {
			t.Errorf("resolved label P0 leaked into the missing set: %v", msg)
		}
	})

	t.Run("empty request resolves to empty ids", func(t *testing.T) {
		ids, err := matchForgejoLabelIDs(table, nil, "px")
		if err != nil || len(ids) != 0 {
			t.Errorf("ids=%v err=%v, want empty,nil", ids, err)
		}
	})
}
