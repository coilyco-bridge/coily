package main

import (
	"encoding/json"
	"strings"
	"testing"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/cli/passthrough"
	"github.com/urfave/cli/v3"
)

// TestPassthroughUsagePrefixMatchesCLIGuard pins the structural contract
// --tree --json depends on: cli-guard's passthrough.Command stamps a Usage
// string our passthroughUsagePrefix is a prefix of, and the leaf Name is
// the bin. If cli-guard reworks that string this fails here instead of
// silently dropping the bin mapping from the generated hook surfaces.
func TestPassthroughUsagePrefixMatchesCLIGuard(t *testing.T) {
	r := newTestRunner(t)
	cmd := passthrough.Command("kubectl", r.Runner, r.Audit)
	if !strings.HasPrefix(cmd.Usage, passthroughUsagePrefix) {
		t.Fatalf("passthrough Usage %q lost prefix %q", cmd.Usage, passthroughUsagePrefix)
	}
	bin, ok := passthroughBin(cmd)
	if !ok || bin != "kubectl" {
		t.Fatalf("passthroughBin = (%q,%v), want (kubectl,true)", bin, ok)
	}
}

// TestPassthroughBin_RejectsNonPassthrough proves a normal verb that shares
// a name with a binary is not mistagged: only the Usage prefix promotes a
// node to a passthrough leaf.
func TestPassthroughBin_RejectsNonPassthrough(t *testing.T) {
	c := &cli.Command{Name: "kubectl", Usage: "Some hand-written verb."}
	if bin, ok := passthroughBin(c); ok {
		t.Errorf("passthroughBin tagged a non-passthrough as bin %q", bin)
	}
}

// TestTreeNodeOf_SkipsHiddenAndHelp mirrors the human printCmdTree filter:
// hidden nodes and the auto-injected help verb never appear in the JSON.
func TestTreeNodeOf_SkipsHiddenAndHelp(t *testing.T) {
	root := &cli.Command{
		Name: "parent",
		Commands: []*cli.Command{
			{Name: "visible"},
			{Name: "secret", Hidden: true},
			{Name: "help"},
		},
	}
	n := treeNodeOf(root)
	if len(n.Children) != 1 || n.Children[0].Name != "visible" {
		t.Fatalf("children = %+v, want only [visible]", n.Children)
	}
}

// TestBuildTree_TagsPassthroughLeaves builds the real coily command tree
// and asserts the ops passthroughs carry their bin mapping while group
// nodes do not.
func TestBuildTree_TagsPassthroughLeaves(t *testing.T) {
	r := newTestRunner(t)
	root := buildTree(r.builtInCommands(), nil)

	ops := childByName(root, "ops")
	if ops == nil {
		t.Fatal("no ops node in tree")
	}
	if ops.Bin != "" {
		t.Errorf("ops group node carries bin %q, want empty", ops.Bin)
	}
	kubectl := childByName(ops, "kubectl")
	if kubectl == nil {
		t.Fatal("no ops kubectl leaf")
	}
	if kubectl.Bin != "kubectl" {
		t.Errorf("ops kubectl bin = %q, want kubectl", kubectl.Bin)
	}
}

// TestFindSubtree covers descent, the empty-path (root) case, and a miss.
func TestFindSubtree(t *testing.T) {
	r := newTestRunner(t)
	root := buildTree(r.builtInCommands(), nil)

	if got := findSubtree(root, nil); got != root {
		t.Errorf("empty path = %v, want root", got)
	}
	if got := findSubtree(root, []string{"ops", "kubectl"}); got == nil || got.Bin != "kubectl" {
		t.Errorf("path ops/kubectl = %+v, want kubectl leaf", got)
	}
	if got := findSubtree(root, []string{"ops", "nope"}); got != nil {
		t.Errorf("path ops/nope = %+v, want nil", got)
	}
}

// TestParseTreeJSONRequest is the safety-critical parser: it must catch the
// leading-flag forms and must NOT hijack a passthrough's forwarded argv.
func TestParseTreeJSONRequest(t *testing.T) {
	cases := []struct {
		name     string
		argv     []string
		wantOK   bool
		wantPath []string
	}{
		{"full tree", []string{"coily", "--tree", "--json"}, true, nil},
		{"flags reversed", []string{"coily", "--json", "--tree"}, true, nil},
		{"subtree path", []string{"coily", "--tree", "--json", "ops"}, true, []string{"ops"}},
		{"deep path", []string{"coily", "--tree", "--json", "ops", "kubectl"}, true, []string{"ops", "kubectl"}},
		{"eq form", []string{"coily", "--tree=true", "--json=true", "ops"}, true, []string{"ops"}},
		{"tree only", []string{"coily", "--tree"}, false, nil},
		{"json only", []string{"coily", "--json"}, false, nil},
		{"passthrough not hijacked", []string{"coily", "ops", "gh", "--tree", "--json"}, false, nil},
		{"pkg passthrough not hijacked", []string{"coily", "pkg", "npm", "--tree", "--json"}, false, nil},
		{"bare", []string{"coily"}, false, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, ok := parseTreeJSONRequest(tc.argv)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && strings.Join(path, ",") != strings.Join(tc.wantPath, ",") {
				t.Errorf("path = %v, want %v", path, tc.wantPath)
			}
		})
	}
}

// TestTreeNode_JSONShape locks the wire format the generators parse: bin is
// omitted on group nodes and present on passthrough leaves.
func TestTreeNode_JSONShape(t *testing.T) {
	n := &treeNode{Name: "ops", Children: []*treeNode{
		{Name: "kubectl", Usage: "Pass-through to kubectl with argv validation + audit log.", Bin: "kubectl"},
	}}
	b, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	if strings.Contains(got, `"bin":"ops"`) || strings.Contains(got, `"bin":""`) {
		t.Errorf("group node leaked a bin field: %s", got)
	}
	if !strings.Contains(got, `"bin":"kubectl"`) {
		t.Errorf("leaf missing bin: %s", got)
	}
}

// childByName returns the direct child with the given name, or nil.
func childByName(n *treeNode, name string) *treeNode {
	for _, c := range n.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}
