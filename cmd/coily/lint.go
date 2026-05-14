package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/coilysiren/cli-guard/exitcode"
	"github.com/coilysiren/cli-guard/repocfg"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// lintCommand validates .coily/coily.yaml against the repo's Makefile so
// the coily-verb surface and the make-target surface cannot drift. Rules:
//   - commands.<verb>.run must equal "make <verb>".
//   - The Makefile must declare a target named <verb>.
//   - The verb description must equal the Makefile target's `## desc`
//     auto-help comment.
func (r *Runner) lintCommand() *cli.Command {
	return &cli.Command{
		Name:  "lint",
		Usage: "Lint .coily/coily.yaml against the repo Makefile.",
		Action: func(_ context.Context, _ *cli.Command) error {
			return runCoilyLint()
		},
	}
}

// makeTargetHelp is the auto-help comment after a Makefile target's colon:
//
//	target: deps  ## description
//
// The description is everything after `## ` to end-of-line, trimmed.
var makeTargetHelp = regexp.MustCompile(`^([A-Za-z0-9_.-]+)\s*:[^=]*?##\s*(.*)$`)

func runCoilyLint() error {
	cwd, err := os.Getwd()
	if err != nil {
		return exitcode.New(exitcode.Internal, "internal", err, "")
	}
	yamlPath, err := repocfg.Discover(cwd)
	if err != nil {
		return exitcode.New(exitcode.UserError, "user_error", err,
			"run from inside a repo with .coily/coily.yaml")
	}
	repoRoot := filepath.Dir(filepath.Dir(yamlPath))
	makefilePath := filepath.Join(repoRoot, "Makefile")

	verbs, err := loadCoilyYamlVerbs(yamlPath)
	if err != nil {
		return exitcode.New(exitcode.UserError, "user_error", err, "")
	}
	targets, err := loadMakefileTargets(makefilePath)
	if err != nil {
		return exitcode.New(exitcode.UserError, "user_error", err, "")
	}

	var problems []string
	for _, v := range verbs {
		want := "make " + v.name
		if v.run != want {
			problems = append(problems, fmt.Sprintf(
				"%s:%d: commands.%s.run = %q, want %q",
				yamlPath, v.line, v.name, v.run, want))
		}
		t, ok := targets[v.name]
		if !ok {
			problems = append(problems, fmt.Sprintf(
				"%s:%d: commands.%s has no matching Makefile target",
				yamlPath, v.line, v.name))
			continue
		}
		if v.description != t.description {
			problems = append(problems, fmt.Sprintf(
				"%s:%d: commands.%s.description = %q, want %q (from %s:%d)",
				yamlPath, v.line, v.name, v.description, t.description,
				makefilePath, t.line))
		}
	}
	if len(problems) > 0 {
		return exitcode.New(exitcode.UserError, "user_error",
			errors.New(strings.Join(problems, "\n")),
			"align coily verb names + descriptions with the Makefile, or update the Makefile to match")
	}
	fmt.Printf("coily lint: %d verbs OK\n", len(verbs))
	return nil
}

type coilyVerb struct {
	name        string
	run         string
	description string
	line        int
}

// loadCoilyYamlVerbs parses .coily/coily.yaml preserving key order and
// capturing line numbers for error messages.
func loadCoilyYamlVerbs(path string) ([]coilyVerb, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	commands, err := findCommandsNode(path, &root)
	if err != nil {
		return nil, err
	}
	verbs := make([]coilyVerb, 0, len(commands.Content)/2)
	for i := 0; i+1 < len(commands.Content); i += 2 {
		k, v := commands.Content[i], commands.Content[i+1]
		if v.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("%s:%d: commands.%s is not a mapping", path, k.Line, k.Value)
		}
		verbs = append(verbs, parseVerbNode(k, v))
	}
	return verbs, nil
}

// findCommandsNode returns the yaml mapping under the top-level
// `commands:` key, or an error if the document shape is wrong.
func findCommandsNode(path string, root *yaml.Node) (*yaml.Node, error) {
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: top level is not a mapping", path)
	}
	doc := root.Content[0]
	for i := 0; i+1 < len(doc.Content); i += 2 {
		if doc.Content[i].Value != "commands" {
			continue
		}
		commands := doc.Content[i+1]
		if commands.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("%s: 'commands' is not a mapping", path)
		}
		return commands, nil
	}
	return nil, fmt.Errorf("%s: missing top-level 'commands:' map", path)
}

// parseVerbNode extracts run + description from a single command's
// mapping node. Unknown keys are ignored; the loader (pkg/repocfg)
// remains the source of truth for accepted shape.
func parseVerbNode(key, value *yaml.Node) coilyVerb {
	verb := coilyVerb{name: key.Value, line: key.Line}
	for j := 0; j+1 < len(value.Content); j += 2 {
		switch value.Content[j].Value {
		case "run":
			verb.run = value.Content[j+1].Value
		case "description":
			verb.description = value.Content[j+1].Value
		}
	}
	return verb
}

type makeTarget struct {
	name        string
	description string
	line        int
}

// loadMakefileTargets returns every target declared with a `## desc`
// auto-help comment. Targets without one are not surfaced; the linter's
// description-match rule treats them as missing.
func loadMakefileTargets(path string) (map[string]makeTarget, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	defer f.Close()
	out := make(map[string]makeTarget)
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		m := makeTargetHelp.FindStringSubmatch(scanner.Text())
		if m == nil {
			continue
		}
		out[m[1]] = makeTarget{name: m[1], description: strings.TrimSpace(m[2]), line: lineNo}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return out, nil
}
