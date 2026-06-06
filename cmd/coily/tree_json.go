package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"
)

// passthroughUsagePrefix is the deterministic Usage string cli-guard's
// passthrough.Command stamps on every wrapper leaf ("Pass-through to gh
// with argv validation + audit log."). It is the structural signal that a
// tree node fronts an underlying binary: the leaf's Name is the bin and
// this prefix flags it, so --tree --json can enumerate the bin mapping
// (`ops kubectl -> kubectl`) without a second registry to drift against.
// If cli-guard changes that string, treeNodeOf stops tagging Bin and the
// TestPassthroughUsagePrefixMatchesCLIGuard guard fails loudly.
const passthroughUsagePrefix = "Pass-through to "

// treeNode is the machine-readable shape of one command-tree node emitted
// by `coily --tree --json`. Passthrough leaves carry Bin (the underlying
// binary they front); group and built-in nodes leave it empty. The
// generated hook surfaces (PreToolUse recovery, SessionStart capability
// index) consume this so coily stays the single source of truth for what
// the agent can reach.
type treeNode struct {
	Name     string      `json:"name"`
	Usage    string      `json:"usage,omitempty"`
	Bin      string      `json:"bin,omitempty"`
	Children []*treeNode `json:"children,omitempty"`
}

// treeNodeOf converts one cli.Command (and its subtree) into a treeNode,
// skipping hidden nodes and the auto-injected `help` verb the same way the
// human printCmdTree does. Returns nil for a skipped node so callers can
// filter it out.
func treeNodeOf(c *cli.Command) *treeNode {
	if c.Hidden || c.Name == "help" {
		return nil
	}
	n := &treeNode{Name: c.Name, Usage: c.Usage}
	if bin, ok := passthroughBin(c); ok {
		n.Bin = bin
	}
	for _, child := range c.Commands {
		if cn := treeNodeOf(child); cn != nil {
			n.Children = append(n.Children, cn)
		}
	}
	return n
}

// passthroughBin reports whether c is a cli-guard passthrough wrapper and,
// if so, the underlying binary it fronts. The binary is the node's Name
// (passthrough.Command sets Name: bin); the Usage prefix is the discriminator
// so a non-passthrough verb that happens to share a name is never mistagged.
func passthroughBin(c *cli.Command) (string, bool) {
	if strings.HasPrefix(c.Usage, passthroughUsagePrefix) {
		return c.Name, true
	}
	return "", false
}

// buildTree assembles the full coily command tree (built-ins + the repo
// `exec` subtree) into a single synthetic root node named "coily". Mirrors
// the surfaces treeCommand renders for humans, but as data.
func buildTree(builtIns []*cli.Command, exec *cli.Command) *treeNode {
	root := &treeNode{Name: "coily"}
	for _, c := range builtIns {
		if cn := treeNodeOf(c); cn != nil {
			root.Children = append(root.Children, cn)
		}
	}
	if exec != nil {
		if cn := treeNodeOf(exec); cn != nil {
			root.Children = append(root.Children, cn)
		}
	}
	return root
}

// findSubtree descends root along the space/segment path (["ops"],
// ["ops", "gh"]) and returns the matched node. An empty path returns root.
// A path that does not resolve returns nil, so the caller can emit a
// not-found error instead of an empty object.
func findSubtree(root *treeNode, path []string) *treeNode {
	node := root
	for _, seg := range path {
		var next *treeNode
		for _, child := range node.Children {
			if child.Name == seg {
				next = child
				break
			}
		}
		if next == nil {
			return nil
		}
		node = next
	}
	return node
}

// parseTreeJSONRequest detects a root-level `coily --tree --json [path...]`
// invocation and returns the subtree path. It returns ok=false unless BOTH
// --tree and --json appear among the LEADING flags - the run of tokens
// before the first non-flag token. That leading-only rule is the safety
// property: a passthrough forwards its argv verbatim, so `coily ops gh
// --tree --json` must NOT be hijacked. There the leading token is `ops`
// (not a flag), so the flags land in the path tail, not the leading set,
// and ok is false. argv is the full process argv (argv[0] is the program).
func parseTreeJSONRequest(argv []string) (path []string, ok bool) {
	if len(argv) < 2 {
		return nil, false
	}
	var sawTree, sawJSON bool
	i := 1
	for ; i < len(argv); i++ {
		tok := argv[i]
		if !strings.HasPrefix(tok, "-") {
			break // first non-flag token: the rest is the path
		}
		switch tok {
		case "--tree", "-tree", "--tree=true", "-tree=true":
			sawTree = true
		case "--json", "-json", "--json=true", "-json=true":
			sawJSON = true
		}
	}
	if !sawTree || !sawJSON {
		return nil, false
	}
	return argv[i:], true
}

// treeJSONCommand renders the command tree as JSON, optionally scoped to a
// subtree named by path (`coily --tree --json ops`). Foundation for the
// generated hook surfaces (coilyco-bridge/coily#197): a machine-readable
// tree where each passthrough leaf carries its underlying-binary mapping.
func treeJSONCommand(builtIns []*cli.Command, exec *cli.Command, _ repoExecResult, path []string) error {
	root := buildTree(builtIns, exec)
	node := findSubtree(root, path)
	if node == nil {
		return cli.Exit(fmt.Sprintf("coily --tree --json: no command at path %q", strings.Join(path, " ")), 1)
	}
	out, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		return fmt.Errorf("coily --tree --json: marshal: %w", err)
	}
	fmt.Println(string(out))
	return nil
}
