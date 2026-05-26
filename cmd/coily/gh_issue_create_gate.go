package main

import (
	"strings"

	"github.com/urfave/cli/v3"
)

// Preflight gate: reject gh invocations that create a GitHub issue (Kai uses Forgejo).
func ghIssueCreateGate(argv []string) error {
	if !isGHIssueCreate(argv) {
		return nil
	}
	return cli.Exit(ghIssueCreateBlockMessage, 1)
}

func isGHIssueCreate(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	switch argv[0] {
	case "issue":
		return len(argv) > 1 && (argv[1] == "create" || argv[1] == "new")
	case "api":
		return isPostToIssuesCollection(argv[1:])
	}
	return false
}

func isPostToIssuesCollection(rest []string) bool {
	method, hasField, path := parseGHAPIArgs(rest)
	if path == "" || !isIssuesCollectionPath(path) {
		return false
	}
	if method == "POST" {
		return true
	}
	// `gh api` defaults to POST when -f / -F fields are present.
	return method == "" && hasField
}

func parseGHAPIArgs(rest []string) (method string, hasField bool, path string) {
	for i := 0; i < len(rest); i++ {
		tok := rest[i]
		if m, consumed := ghAPIMethodFlag(tok, rest, i); m != "" || consumed > 0 {
			if m != "" {
				method = m
			}
			i += consumed
			continue
		}
		if ghAPIFieldFlag(tok) {
			hasField = true
			if i+1 < len(rest) {
				i++
			}
			continue
		}
		if strings.HasPrefix(tok, "-") {
			continue
		}
		if path == "" {
			path = tok
		}
	}
	return method, hasField, path
}

func ghAPIMethodFlag(tok string, rest []string, i int) (method string, consumed int) {
	switch {
	case tok == "-X" || tok == "--method":
		if i+1 < len(rest) {
			return strings.ToUpper(rest[i+1]), 1
		}
		return "", 1
	case strings.HasPrefix(tok, "-X"):
		return strings.ToUpper(strings.TrimPrefix(tok, "-X")), 0
	case strings.HasPrefix(tok, "--method="):
		return strings.ToUpper(strings.TrimPrefix(tok, "--method=")), 0
	}
	return "", 0
}

func ghAPIFieldFlag(tok string) bool {
	return tok == "-f" || tok == "-F" || tok == "--field" || tok == "--raw-field"
}

func isIssuesCollectionPath(path string) bool {
	trimmed := strings.TrimSuffix(path, "/")
	segs := strings.Split(trimmed, "/")
	if len(segs) == 0 {
		return false
	}
	return segs[len(segs)-1] == "issues"
}

const ghIssueCreateBlockMessage = `coily: GitHub issue creation is blocked. Kai migrated TODOs to Forgejo.

Blocked: ` + "`gh issue create`" + `, ` + "`gh issue new`" + `, and ` + "`gh api`" + ` POST to
` + "`/repos/<owner>/<repo>/issues`" + `. Reads, edits, closes, and comments
still pass through ` + "`coily ops gh`" + ` untouched.

File the issue on Forgejo instead:
  coily ops forgejo issue create --repo coilysiren/<name> --title ... --body-file /tmp/body.md

See agentic-os-kai AGENTS.md "Default TODO Destination".`
