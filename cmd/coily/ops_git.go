package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/coilysiren/cli-guard/audit"
	"github.com/coilysiren/cli-guard/decision"
	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// gitCommand groups the git verbs coily fronts: the audit-trailer
// integration (trailer, trailer-hook, audit-show) plus thin passthroughs
// for the heavy-rotation git commands (status, log, diff, pull, push, ...).
//
// The audit-trailer subcommands are SkipScope: they're meta - operating on
// the audit log itself - and the prepare-commit-msg hook calls `coily git
// trailer` from inside the repo regardless of the global --commit-scope.
// The passthrough subcommands bind scope like any other passthrough.
func (r *Runner) gitCommand() *cli.Command {
	return &cli.Command{
		Name:  "git",
		Usage: "git passthroughs plus audit-log integration for commits.",
		Description: `git fronts the everyday git commands (status, log, diff, add, commit,
fetch, pull, push, branch, checkout, stash, restore) as audited
passthroughs, and carries the audit-trailer integration: 'trailer'
emits the Audit-log: lines for the commit being authored, 'audit-show'
resolves a trailer back to its underlying record.`,
		Commands: append([]*cli.Command{
			r.gitTrailerCommand(),
			r.gitAuditShowCommand(),
			r.gitTrailerHookCommand(),
		}, r.gitPassthroughCommands()...),
	}
}

// gitTrailerHookCommand is the prepare-commit-msg implementation. Pre-commit
// framework invokes it as `coily git trailer-hook <commit-msg-file>
// [<source>] [<sha1>]`. Skips merge/squash commits, otherwise appends the
// 'coily git trailer' output to the commit message in place. Exits non-zero
// on any error so the commit is blocked - matches the "block always" mode
// Kai picked when designing the workflow.
func (r *Runner) gitTrailerHookCommand() *cli.Command {
	return &cli.Command{
		Name:      "trailer-hook",
		Usage:     "prepare-commit-msg hook: append Audit-log: trailers in place.",
		ArgsUsage: "<commit-msg-file> [<source> [<sha1>]]",
		Description: `trailer-hook is wired into a repo as a prepare-commit-msg hook (via
pre-commit framework or .git/hooks/prepare-commit-msg). Reads the commit
message file, computes trailers for the current repo, and rewrites the
file with the trailers appended.

Skipped sources: merge, squash. Anything else (the default 'message'
source, plus 'template' / 'commit') gets trailers.

Exits non-zero on any error - missing audit log, scope can't be
resolved, write fails. The commit is then blocked until the operator
fixes the cause or bypasses with --no-verify (which they won't, per
the global git workflow rule).`,
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "git.trailer-hook",
				SkipScope: true,
				Action: func(_ context.Context, c *cli.Command) error {
					return runTrailerHook(r, c)
				},
			},
			r.Audit,
		),
	}
}

func runTrailerHook(r *Runner, c *cli.Command) error {
	args := c.Args().Slice()
	if len(args) == 0 {
		return fmt.Errorf("git trailer-hook: missing <commit-msg-file> argument")
	}
	msgFile := args[0]
	source := ""
	if len(args) >= 2 {
		source = args[1]
	}
	if source == "merge" || source == "squash" {
		// Don't touch merge/squash messages: too easy to confuse the trailer
		// block when git is also auto-stuffing conflict markers / squash
		// summaries. Acceptable: merge commits aren't where the chain of
		// trust lives.
		return nil
	}

	scopePath, err := resolveHookScope(msgFile)
	if err != nil {
		return err
	}
	body, err := os.ReadFile(msgFile) //nolint:gosec // path supplied by git via prepare-commit-msg
	if err != nil {
		return fmt.Errorf("git trailer-hook: read %s: %w", msgFile, err)
	}
	if hasAuditTrailer(body) {
		return nil
	}

	since, _ := resolveSince("", scopePath)
	records, err := loadAuditRecords(r.Cfg.Audit.LogPath, scopePath, since)
	if err != nil {
		return err
	}
	merged := mergeTrailerBlock(body, renderTrailerBlock(records))
	if err := os.WriteFile(msgFile, []byte(merged), 0o644); err != nil { //nolint:gosec // commit msg files are user-readable by design
		return fmt.Errorf("git trailer-hook: write %s: %w", msgFile, err)
	}
	return nil
}

func resolveHookScope(msgFile string) (string, error) {
	scopePath, err := gitToplevel(filepath.Dir(msgFile))
	if err == nil {
		return scopePath, nil
	}
	// .git/COMMIT_EDITMSG should always be inside a repo; if dir-of-msg
	// doesn't resolve, fall back to cwd.
	alt, fallbackErr := gitToplevel(".")
	if fallbackErr != nil {
		return "", fmt.Errorf("git trailer-hook: cannot resolve repo from %s or cwd: %w", msgFile, err)
	}
	return alt, nil
}

func renderTrailerBlock(records []audit.Record) string {
	var b strings.Builder
	if len(records) == 0 {
		fmt.Fprintf(&b, "%s: %s\n", trailerKey, trailerNoneValue)
		return b.String()
	}
	for _, rec := range records {
		t := rec.TrailerLine()
		if t == "" {
			continue
		}
		fmt.Fprintf(&b, "%s: %s\n", trailerKey, t)
	}
	return b.String()
}

// hasAuditTrailer is a simple guard against double-appending if the hook
// fires twice on the same message.
func hasAuditTrailer(body []byte) bool {
	for _, line := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, trailerKey+":") {
			return true
		}
	}
	return false
}

// mergeTrailerBlock inserts the rendered Audit-log: lines into the body.
// Strategy: append at end, separated from existing content by a blank line.
// Git's trailer parser collects all trailing 'Key: value' lines as one
// block, so a single blank-line separation is enough.
func mergeTrailerBlock(body []byte, trailers string) string {
	s := strings.TrimRight(string(body), "\n")
	if s == "" {
		return trailers
	}
	// Detect whether the message already ends with a trailer-shaped block
	// (Co-Authored-By, Signed-off-by, etc). If so, append directly without
	// the blank line - the existing block becomes a single contiguous
	// trailer section.
	lines := strings.Split(s, "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if isTrailerLine(last) {
		return s + "\n" + trailers
	}
	return s + "\n\n" + trailers
}

func isTrailerLine(line string) bool {
	if line == "" {
		return false
	}
	colon := strings.IndexByte(line, ':')
	if colon <= 0 {
		return false
	}
	key := line[:colon]
	for _, r := range key {
		if r != '-' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return false
		}
	}
	return true
}

const (
	defaultTrailerMax     = 20
	defaultTrailerWindow  = time.Hour
	trailerKey            = "Audit-log"
	trailerNoneValue      = "none"
	trailerTruncatedFmt   = "%d earlier rows truncated; run `coily git audit-show --since=%d --scope=%s` for full list"
	trailerHookEnvWarning = "COILY_TRAILER_HOOK"
)

func (r *Runner) gitTrailerCommand() *cli.Command {
	return &cli.Command{
		Name:  "trailer",
		Usage: "Emit Audit-log: trailers for the current repo.",
		Description: `trailer scans the audit log for rows where commit_scope matches the
target repo and ts is at or after --since, then prints one
'Audit-log: coily://<ts>/<short-id>' line per row (capped at --max).
If no rows match, prints 'Audit-log: none' so a downstream hook can
treat empty as "explicitly nothing" rather than "tool failed."

Designed to be invoked from a prepare-commit-msg hook. Output goes to
stdout; the hook is expected to append it to the commit message.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "scope",
				Usage: "repo path to filter by (default: git toplevel of cwd)",
			},
			&cli.StringFlag{
				Name:  "since",
				Usage: "unix seconds or duration (`5m`, `1h`); default: timestamp of HEAD commit, or 1h if no HEAD",
			},
			&cli.IntFlag{
				Name:  "max",
				Usage: "maximum number of trailer lines (excess are summarized)",
				Value: defaultTrailerMax,
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "git.trailer",
				SkipScope: true,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--scope": c.String("scope"),
						"--since": c.String("since"),
					}, nil
				},
				Action: func(_ context.Context, c *cli.Command) error {
					return runTrailer(r, c)
				},
			},
			r.Audit,
		),
	}
}

func runTrailer(r *Runner, c *cli.Command) error {
	scopePath := c.String("scope")
	if scopePath == "" {
		top, err := gitToplevel(".")
		if err != nil {
			return fmt.Errorf("git trailer: --scope unset and cwd is not in a git repo: %w", err)
		}
		scopePath = top
	}
	abs, err := filepath.Abs(scopePath)
	if err != nil {
		return fmt.Errorf("git trailer: abs %s: %w", scopePath, err)
	}
	scopePath = filepath.Clean(abs)

	since, err := resolveSince(c.String("since"), scopePath)
	if err != nil {
		return err
	}
	maxLines := c.Int("max")
	if maxLines <= 0 {
		maxLines = defaultTrailerMax
	}

	records, err := loadAuditRecords(r.Cfg.Audit.LogPath, scopePath, since)
	if err != nil {
		return err
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].Timestamp < records[j].Timestamp
	})

	if len(records) == 0 {
		fmt.Printf("%s: %s\n", trailerKey, trailerNoneValue)
		return nil
	}
	emitted := records
	truncated := 0
	if len(records) > maxLines {
		// Keep the most recent `maxLines` rows; surface the older ones as a
		// summary line so the reader knows there's more.
		truncated = len(records) - maxLines
		emitted = records[truncated:]
	}
	for _, rec := range emitted {
		t := rec.TrailerLine()
		if t == "" {
			continue
		}
		fmt.Printf("%s: %s\n", trailerKey, t)
	}
	if truncated > 0 {
		fmt.Printf("%s: "+trailerTruncatedFmt+"\n", trailerKey, truncated, since, scopePath)
	}
	return nil
}

func resolveSince(raw, repoPath string) (int64, error) {
	if raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return n, nil
		}
		d, err := time.ParseDuration(raw)
		if err != nil {
			return 0, fmt.Errorf("git trailer: --since must be unix seconds or a duration: %w", err)
		}
		return time.Now().Add(-d).Unix(), nil
	}
	// Fallback: HEAD commit's committer timestamp. If HEAD is unborn (fresh
	// branch / no commits) or git fails, default to the last hour.
	out, gitErr := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%ct", "HEAD").Output()
	if gitErr != nil {
		// Fresh branch / unborn HEAD: fall back to a time-window default.
		return time.Now().Add(-defaultTrailerWindow).Unix(), nil //nolint:nilerr // see comment
	}
	s := strings.TrimSpace(string(out))
	n, parseErr := strconv.ParseInt(s, 10, 64)
	if parseErr != nil {
		return time.Now().Add(-defaultTrailerWindow).Unix(), nil //nolint:nilerr // see comment
	}
	return n, nil
}

// scopeMatches returns true when a recorded commit-scope is the filter
// scope itself or any descendant directory. The boundary check prevents
// "/foo" from matching "/foobar".
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
		return nil, fmt.Errorf("git trailer: open audit log: %w", err)
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
		if !scopeMatches(filepath.Clean(rec.CommitScope), scopePath) {
			continue
		}
		if rec.Decision != audit.DecisionAccept {
			// Reject rows record refused invocations - those didn't actually
			// run anything privileged, so they don't belong on a commit.
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

// gitAuditShowCommand resolves a coily:// trailer back to its audit row,
// printed as yaml.
func (r *Runner) gitAuditShowCommand() *cli.Command {
	return &cli.Command{
		Name:      "audit-show",
		Usage:     "Resolve an Audit-log trailer back to its full audit record.",
		ArgsUsage: "<coily://ts/short-id>",
		Description: `audit-show takes a single coily:// trailer value (the part after
'Audit-log:'), finds the matching audit row by exact (timestamp,
short-id) match, and prints it as yaml.

Use --since-list to resolve a flag-truncated trailer summary instead -
prints the full set of rows in the same window/scope as the original
'coily git trailer' invocation.`,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "since",
				Usage: "(with --scope) unix seconds; print every row in scope at or after this time",
			},
			&cli.StringFlag{
				Name:  "scope",
				Usage: "(with --since) repo path to filter by",
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "git.audit-show",
				SkipScope: true,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--scope": c.String("scope"),
						"--since": fmt.Sprintf("%d", c.Int("since")),
					}, c.Args().Slice()
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
	if since != 0 && scopePath != "" {
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

	args := c.Args().Slice()
	if len(args) == 0 {
		return fmt.Errorf("git audit-show: pass a coily:// trailer value or --since/--scope")
	}
	trailer := args[0]
	ts, short, ok := audit.ParseTrailer(trailer)
	if !ok {
		return fmt.Errorf("git audit-show: not a valid coily:// trailer: %q", trailer)
	}
	rec, err := findRecordByShortID(r.Cfg.Audit.LogPath, ts, short)
	if err != nil {
		if errors.Is(err, errNotFound) {
			return fmt.Errorf("git audit-show: no audit record matches %s", trailer)
		}
		return err
	}
	return writeRecordsYAML(os.Stdout, []audit.Record{*rec})
}

// ErrNotFound is returned by findRecordByShortID when no audit row matches.
var errNotFound = errors.New("audit record not found")

func findRecordByShortID(path string, ts int64, short string) (*audit.Record, error) {
	f, err := os.Open(path) //nolint:gosec // audit path is config-resolved
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errNotFound
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()
	dec := json.NewDecoder(f)
	for dec.More() {
		var rec audit.Record
		if decErr := dec.Decode(&rec); decErr != nil {
			continue
		}
		if rec.Timestamp != ts {
			continue
		}
		if rec.ShortID() != short {
			continue
		}
		return &rec, nil
	}
	return nil, errNotFound
}

// writeRecordsYAML emits each record as a `trailer:` + `record:` block,
// matching the trailer shape carried by audit.Record's Trailer/TrailerLine
// helpers (see cli-guard/audit).
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
		safeRemoteArgv := audit.RedactArgv(rec.RemoteArgv, audit.DataSecurityHigh, policy)
		view := map[string]any{
			"trailer": rec.Trailer(),
			"record": map[string]any{
				"id":           rec.ID,
				"ts":           rec.Timestamp,
				"ts_iso":       time.Unix(rec.Timestamp, 0).UTC().Format(time.RFC3339),
				"verb":         rec.Verb,
				"argv":         safeArgv,
				"remote_argv":  safeRemoteArgv,
				"decision":     rec.Decision,
				"exit_code":    rec.ExitCode,
				"duration_ms":  rec.DurationMS,
				"repo_root":    rec.RepoRoot,
				"commit_scope": rec.CommitScope,
				"version":      rec.Version,
				"error":        rec.Error,
				"stderr_tail":  rec.StderrTail,
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
