package main

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/coilysiren/cli-guard/audit"
	"github.com/coilysiren/cli-guard/decision"
	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

//go:embed dashboard.html.tmpl
var dashboardTemplate string

// auditDashboardCommand renders a static HTML view of the audit log,
// aggregated by verb. See coilysiren/coily#137 for the design.
func (r *Runner) auditDashboardCommand() *cli.Command {
	return &cli.Command{
		Name:  "dashboard",
		Usage: "Render a static HTML aggregation of the audit log.",
		Description: `dashboard reads the resolved audit log, filters to the --since window,
groups rows by verb, and writes a single self-contained HTML file with
the aggregation data baked in and small embedded JS for sort + filter.

No external network requests. No CDN deps. Open the printed path in a
browser.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "since",
				Usage: "window (duration like 24h, 7d, or unix seconds)",
				Value: "24h",
			},
			&cli.StringFlag{
				Name:  "out",
				Usage: "output HTML path (default: ~/.coily/dashboard.html)",
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "audit.dashboard",
				SkipScope: true,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--since": c.String("since"),
						"--out":   c.String("out"),
					}, nil
				},
				Action: func(_ context.Context, c *cli.Command) error {
					return runAuditDashboard(r, c)
				},
			},
			r.Audit,
		),
	}
}

func runAuditDashboard(r *Runner, c *cli.Command) error {
	since, err := parseSince(c.String("since"))
	if err != nil {
		return err
	}
	if since == 0 {
		// parseSince returns 0 for empty input. For dashboard the empty case
		// would render every row ever, which is rarely what the operator
		// wants and can be slow. Default to 24h instead.
		since = time.Now().Add(-24 * time.Hour).Unix()
	}

	records, err := readRecords(r.Cfg.Audit.LogPath, since)
	if err != nil {
		return err
	}
	view := buildDashboardView(records, since)

	outPath := c.String("out")
	if outPath == "" {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return fmt.Errorf("audit dashboard: resolve home: %w", herr)
		}
		outPath = filepath.Join(home, ".coily", "dashboard.html")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("audit dashboard: mkdir %s: %w", filepath.Dir(outPath), err)
	}
	if err := renderDashboard(outPath, view); err != nil {
		return err
	}
	fmt.Println(outPath)
	return nil
}

// readRecords reads the JSONL audit log, skipping malformed lines and
// records older than since.
func readRecords(path string, since int64) ([]audit.Record, error) {
	f, err := os.Open(path) //nolint:gosec // resolved via cli-guard/config; reading is the point
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("audit dashboard: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	var out []audit.Record
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		var rec audit.Record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		if rec.Timestamp < since {
			continue
		}
		out = append(out, rec)
	}
	if err := s.Err(); err != nil {
		return out, fmt.Errorf("audit dashboard: scan %s: %w", path, err)
	}
	return out, nil
}

// dashboardView is the shape the template renders.
type dashboardView struct {
	GeneratedAt  string       `json:"generated_at"`
	WindowStart  string       `json:"window_start"`
	WindowSince  int64        `json:"window_since"`
	TotalRecords int          `json:"total_records"`
	Verbs        []verbBucket `json:"verbs"`
}

// verbBucket is one aggregated verb row.
type verbBucket struct {
	Verb        string   `json:"verb"`
	Total       int      `json:"total"`
	Accepted    int      `json:"accepted"`
	Rejected    int      `json:"rejected"`
	Failed      int      `json:"failed"`
	Repos       []string `json:"repos"`
	EgressHosts []string `json:"egress_hosts"`
	ArgvSamples []string `json:"argv_samples"`
	LastRunUnix int64    `json:"last_run_unix"`
	LastRunISO  string   `json:"last_run_iso"`
	TotalEgress int64    `json:"total_egress_bytes"`
}

const argvSampleCap = 5

// verbAcc accumulates one verb's running aggregate.
type verbAcc struct {
	bucket   verbBucket
	repoSet  map[string]struct{}
	hostSet  map[string]struct{}
	argvSeen map[string]struct{}
}

func newVerbAcc(verb string) *verbAcc {
	return &verbAcc{
		bucket:   verbBucket{Verb: verb},
		repoSet:  map[string]struct{}{},
		hostSet:  map[string]struct{}{},
		argvSeen: map[string]struct{}{},
	}
}

// add folds one audit record into this verb's accumulator.
func (a *verbAcc) add(r audit.Record) {
	a.bucket.Total++
	a.countDecision(r)
	if r.CommitScope != "" {
		a.repoSet[filepath.Base(r.CommitScope)] = struct{}{}
	}
	for _, e := range r.Egress {
		if e.Host == "" {
			continue
		}
		a.hostSet[e.Host] = struct{}{}
		a.bucket.TotalEgress += e.BytesUp + e.BytesDown
	}
	if r.Timestamp > a.bucket.LastRunUnix {
		a.bucket.LastRunUnix = r.Timestamp
	}
	a.addArgvSample(r.Argv)
}

func (a *verbAcc) countDecision(r audit.Record) {
	switch r.Decision {
	case audit.DecisionAccept:
		a.bucket.Accepted++
		if r.ExitCode != 0 {
			// Accepted-then-failed: the gate let it through but the wrapped
			// tool exited non-zero. Surface as failed too.
			a.bucket.Failed++
		}
	case audit.DecisionReject:
		a.bucket.Rejected++
	default:
		if r.ExitCode != 0 {
			a.bucket.Failed++
		}
	}
}

func (a *verbAcc) addArgvSample(argv []string) {
	if len(a.bucket.ArgvSamples) >= argvSampleCap {
		return
	}
	// Cross-surface redaction (coily#252, forward action for finding
	// #228): the dashboard is a less-restricted surface than the on-disk
	// JSONL, so secret-flag values get redacted before joining
	// regardless of the row's stored data_security tier. Preserves the
	// verb path + non-secret flags so sample diversity still informs
	// the dashboard.
	safe := audit.RedactArgv(argv, audit.DataSecurityHigh, decision.RedactPolicy())
	joined := strings.Join(safe, " ")
	if _, seen := a.argvSeen[joined]; seen {
		return
	}
	a.argvSeen[joined] = struct{}{}
	a.bucket.ArgvSamples = append(a.bucket.ArgvSamples, joined)
}

func (a *verbAcc) finalize() verbBucket {
	a.bucket.Repos = sortedKeys(a.repoSet)
	a.bucket.EgressHosts = sortedKeys(a.hostSet)
	if a.bucket.LastRunUnix > 0 {
		a.bucket.LastRunISO = time.Unix(a.bucket.LastRunUnix, 0).UTC().Format(time.RFC3339)
	}
	return a.bucket
}

// buildDashboardView groups records by Verb and rolls up the per-verb
// stats. Sorted by Total desc, then Verb asc, so the heaviest hitters
// appear first.
func buildDashboardView(records []audit.Record, since int64) dashboardView {
	buckets := map[string]*verbAcc{}
	for _, r := range records {
		a, ok := buckets[r.Verb]
		if !ok {
			a = newVerbAcc(r.Verb)
			buckets[r.Verb] = a
		}
		a.add(r)
	}

	out := make([]verbBucket, 0, len(buckets))
	for _, a := range buckets {
		out = append(out, a.finalize())
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Total != out[j].Total {
			return out[i].Total > out[j].Total
		}
		return out[i].Verb < out[j].Verb
	})

	return dashboardView{
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		WindowStart:  time.Unix(since, 0).UTC().Format(time.RFC3339),
		WindowSince:  since,
		TotalRecords: len(records),
		Verbs:        out,
	}
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// renderDashboard executes the embedded template against view and writes
// the result to path. The template bakes the view in as a JSON blob, so
// the rendered HTML needs no external fetches.
func renderDashboard(path string, view dashboardView) error {
	blob, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return fmt.Errorf("audit dashboard: marshal view: %w", err)
	}
	t, err := template.New("dashboard").Parse(dashboardTemplate)
	if err != nil {
		return fmt.Errorf("audit dashboard: parse template: %w", err)
	}
	f, err := os.Create(path) //nolint:gosec // path is user-supplied or derived from --out
	if err != nil {
		return fmt.Errorf("audit dashboard: create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return t.Execute(f, map[string]any{
		// G203: template.JS marks the bytes as pre-trusted JS. Safe here
		// because `blob` is json.Marshal of a structured dashboardView whose
		// fields originate from the audit log on disk, not from a network
		// request or user-supplied HTML. The renderer above uses textContent
		// for every Argv string at insertion time, so even if a future
		// caller passes attacker-controlled argv, it cannot escape the DOM.
		"DataJSON":     template.JS(blob), //nolint:gosec // see comment above
		"GeneratedAt":  view.GeneratedAt,
		"WindowStart":  view.WindowStart,
		"TotalRecords": view.TotalRecords,
		"VerbCount":    len(view.Verbs),
	})
}
