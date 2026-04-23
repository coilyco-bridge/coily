package main

import (
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/verbclass"
)

// TestBuildTreeAssignsScopes is the smoke test for the leaf-classification
// pass. Verb classification itself lives in pkg/verbclass and has its own
// table-driven tests; this only checks that buildTree wires the bucket and
// scope onto the right nodes.
func TestBuildTreeAssignsScopes(t *testing.T) {
	m := Manifest{
		Binary: "aws",
		Commands: []ManifestCommand{
			{Path: []string{"route53"}, Help: "Route 53 group", Children: []string{"list-hosted-zones", "delete-hosted-zone", "create-hosted-zone"}},
			{Path: []string{"route53", "list-hosted-zones"}, Help: "List hosted zones."},
			{Path: []string{"route53", "delete-hosted-zone"}, Help: "Delete a hosted zone."},
			{Path: []string{"route53", "create-hosted-zone"}, Help: "Create a hosted zone."},
		},
	}
	root := buildTree(m)
	if len(root.Children) != 1 {
		t.Fatalf("got %d top-level children, want 1", len(root.Children))
	}
	r53 := root.Children[0]
	if r53.Name != "route53" {
		t.Fatalf("top-level name = %q, want route53", r53.Name)
	}
	// Group nodes get neither bucket nor scope.
	if r53.Mutating {
		t.Errorf("group node Mutating = true, want false")
	}
	if r53.Scope != "" {
		t.Errorf("group node Scope = %q, want empty", r53.Scope)
	}

	want := map[string]struct {
		mutating bool
		scope    string
	}{
		"list-hosted-zones":  {false, "aws.route53:read"},
		"create-hosted-zone": {true, "aws.route53:write"},
		"delete-hosted-zone": {true, "aws.route53:delete"},
	}
	for _, child := range r53.Children {
		w, ok := want[child.Name]
		if !ok {
			t.Fatalf("unexpected child %q", child.Name)
		}
		if child.Mutating != w.mutating {
			t.Errorf("%s Mutating = %v, want %v", child.Name, child.Mutating, w.mutating)
		}
		if child.Scope != w.scope {
			t.Errorf("%s Scope = %q, want %q", child.Name, child.Scope, w.scope)
		}
	}
}

// TestRenderEmitsScope spot-checks that the rendered template carries the
// scope through to the generated source. A regression here means an agent's
// token check would silently use the wrong scope.
func TestRenderEmitsScope(t *testing.T) {
	m := Manifest{
		Binary: "aws",
		Commands: []ManifestCommand{
			{Path: []string{"route53"}, Help: "group", Children: []string{"delete-hosted-zone"}},
			{Path: []string{"route53", "delete-hosted-zone"}, Help: "Delete a hosted zone."},
		},
	}
	code, err := render(m)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(code, `"aws.route53:delete"`) {
		t.Errorf("rendered code missing aws.route53:delete scope")
	}
	if !strings.Contains(code, "policy.Mutating") {
		t.Errorf("rendered code missing policy.Mutating for delete leaf")
	}
}

// TestVerbclassIsTheSourceOfTruth is a trivial dependency assertion. If
// pkg/verbclass changes its bucket assignment for a representative leaf,
// gen-passthrough picks it up automatically. Listed explicitly so the
// dependency is visible in test output.
func TestVerbclassIsTheSourceOfTruth(t *testing.T) {
	cases := []struct {
		path []string
		want verbclass.Bucket
	}{
		{[]string{"route53", "list-hosted-zones"}, verbclass.Read},
		{[]string{"route53", "create-hosted-zone"}, verbclass.Write},
		{[]string{"route53", "delete-hosted-zone"}, verbclass.Delete},
	}
	for _, tc := range cases {
		if got := verbclass.Classify(tc.path); got != tc.want {
			t.Errorf("Classify(%v) = %s, want %s (verbclass changed; update gen-passthrough golden)",
				tc.path, got, tc.want)
		}
	}
}
