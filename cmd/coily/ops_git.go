package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/cli/decision"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/cli/verb"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/pkg/audit"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// gitCommand groups the git verbs coily fronts: the audit-log query
// ('audit-show') plus thin passthroughs for the heavy-rotation git
// commands (status, log, diff, pull, push, ...).
//
// audit-show queries by the forensic RepoRoot stamped on each row (git
// toplevel of cwd at invocation time), not a caller-supplied scope.
func (r *Runner) gitCommand() *cli.Command {
	return &cli.Command{
		Name:  "git",
		Usage: "git passthroughs plus audit-log query for commits.",
		Description: `git fronts the everyday git commands (status, log, diff, add, commit,
fetch, pull, push, branch, checkout, stash, restore) as audited
passthroughs, and carries 'audit-show', which queries the audit log
for the rows bound to a repo over a time window.`,
		Commands: append([]*cli.Command{
			r.gitAuditShowCommand(),
			r.gitCommitCommand(),
		}, r.gitPassthroughCommands()...),
	}
}

// scopeMatches returns true when a recorded RepoRoot is the filter scope
// itself or any descendant directory. The boundary check prevents "/foo"
// from matching "/foobar".
func scopeMatches(recScope, filterScope string) bool {
	if recScope == filterScope {
		return true
	}
	return strings.HasPrefix(recScope, filterScope+string(filepath.Separator))
}

func loadAuditRecords(path, scopePath string, since int64) ([]audit.Record, error) {
	f, err := os.Open(path) //nolint:gosec // resolved via cli-guard/config; reading is the point
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("git audit-show: open audit log: %w", err)
	}
	defer func() { _ = f.Close() }()

	dec := json.NewDecoder(f)
	var out []audit.Record
	for dec.More() {
		var rec audit.Record
		if err := dec.Decode(&rec); err != nil {
			// Skip malformed lines rather than abort - audit log is forensic.
			continue
		}
		if rec.Timestamp < since {
			continue
		}
		if rec.RepoRoot == "" || !scopeMatches(filepath.Clean(rec.RepoRoot), scopePath) {
			continue
		}
		if rec.Decision != audit.DecisionAccept {
			// Reject rows record refused invocations - those didn't actually
			// run anything privileged, so they don't belong in the query.
			continue
		}
		out = append(out, rec)
	}
	return out, nil
}

func gitToplevel(cwd string) (string, error) {
	out, err := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	top := strings.TrimSpace(string(out))
	if top == "" {
		return "", fmt.Errorf("empty toplevel")
	}
	return filepath.Clean(top), nil
}

// gitAuditShowCommand prints the audit rows bound to a repo over a time
// window, as yaml.
func (r *Runner) gitAuditShowCommand() *cli.Command {
	return &cli.Command{
		Name:  "audit-show",
		Usage: "Print the audit rows bound to a repo over a time window.",
		Description: `audit-show prints every audit row whose repo_root matches --scope
and whose timestamp is at or after --since, as yaml. Use it to inspect
what privileged commands coily ran in a repo over a window.`,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "since",
				Usage:    "unix seconds; print every row in scope at or after this time",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "scope",
				Usage:    "repo path to filter by",
				Required: true,
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "git.audit-show",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--scope": c.String("scope"),
						"--since": fmt.Sprintf("%d", c.Int("since")),
					}, nil
				},
				Action: func(_ context.Context, c *cli.Command) error {
					return runAuditShow(r, c)
				},
			},
			r.Audit,
		),
	}
}

func runAuditShow(r *Runner, c *cli.Command) error {
	since := int64(c.Int("since"))
	scopePath := c.String("scope")
	abs, err := filepath.Abs(scopePath)
	if err != nil {
		return fmt.Errorf("git audit-show: abs scope: %w", err)
	}
	records, err := loadAuditRecords(r.Cfg.Audit.LogPath, filepath.Clean(abs), since)
	if err != nil {
		return err
	}
	return writeRecordsYAML(os.Stdout, records)
}

// writeRecordsYAML emits each record as a `record:` block.
//
// Cross-surface redaction (coily#252, forward action for finding #228):
// argv is re-published into a less-restricted surface (audit-show output
// is operator-piped anywhere), so secret-flag values get redacted before
// emission regardless of the row's stored data_security tier. The high-
// tier redactor blanks `--value`, `--password`, `--token`, `--secret`,
// etc., per decision.RedactPolicy.
func writeRecordsYAML(w io.Writer, records []audit.Record) error {
	policy := decision.RedactPolicy()
	for _, rec := range records {
		safeArgv := audit.RedactArgv(rec.Argv, audit.DataSecurityHigh, policy)
		view := map[string]any{
			"record": map[string]any{
				"id":          rec.ID,
				"ts":          rec.Timestamp,
				"ts_iso":      time.Unix(rec.Timestamp, 0).UTC().Format(time.RFC3339),
				"verb":        rec.Verb,
				"argv":        safeArgv,
				"decision":    rec.Decision,
				"exit_code":   rec.ExitCode,
				"duration_ms": rec.DurationMS,
				"repo_root":   rec.RepoRoot,
				"version":     rec.Version,
				"error":       rec.Error,
				"stderr_tail": rec.StderrTail,
			},
		}
		out, err := yaml.Marshal(view)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "---"); err != nil {
			return err
		}
		if _, err := w.Write(out); err != nil {
			return err
		}
	}
	return nil
}
