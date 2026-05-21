package main

// gh_actions_web.go reads GitHub Actions run status by parsing the
// server-rendered github.com Actions page, instead of the api.github.com
// REST API.
//
// Why this exists: github.com web requests draw on a rate-limit pool
// that is separate from the api.github.com REST 5000/hr budget. When a
// burst of CI polling exhausts the personal-PAT REST budget, run-status
// reads can still succeed off the web page. This is one of the extra
// budget sources in the tiered fallback designed in coilysiren/coily#289;
// the GitHub App installation token (coily#292) is the other.
//
// This is slice A of #289 (coily#291): the fetch + parse only. Wiring it
// into the gh read path as a fallback tier is slice C (coily#293).
//
// Fragility: this parses server-rendered HTML and will break when GitHub
// reshapes the Actions page. That is acceptable for a fallback tier, not
// a primary. The parser is deliberately tolerant - an unrecognized row is
// skipped, never fatal - and leans on the run anchor's aria-label, the
// single most structured field GitHub renders ("<status phrase>:  Run
// <n> of <workflow>. <title>").

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// webRunsBaseURL is the github.com host the Actions HTML is read from.
// A var, not a const, so tests can point it at an httptest server.
var webRunsBaseURL = "https://github.com"

// ghWebCookieEnv carries an optional github.com session cookie for
// reading the Actions page of a private repo. Public repos need no
// cookie. When the cookie is absent (or stale) and the fetch lands on a
// login redirect, fetchActionsRunsHTML returns an error so the caller
// falls through to the next budget tier.
const ghWebCookieEnv = "COILY_GH_WEB_COOKIE"

// webWorkflowRun mirrors the fields of a REST
// /repos/{owner}/{repo}/actions/runs entry that are recoverable from the
// rendered HTML. Fields the page does not expose (workflow_id,
// updated_at, run_attempt) are intentionally omitted rather than guessed;
// callers that need them must use the REST tier.
type webWorkflowRun struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	DisplayTitle string `json:"display_title"`
	HeadBranch   string `json:"head_branch"`
	HeadSHA      string `json:"head_sha"`
	RunNumber    int    `json:"run_number"`
	Event        string `json:"event"`
	Status       string `json:"status"`
	Conclusion   string `json:"conclusion"`
	HTMLURL      string `json:"html_url"`
	CreatedAt    string `json:"created_at"`
	CheckSuiteID int64  `json:"check_suite_id"`
	Actor        *actor `json:"actor,omitempty"`
}

// actor is the trimmed run-actor object, matching the REST shape closely
// enough that `--jq '.workflow_runs[].actor.login'` keeps working.
type actor struct {
	Login string `json:"login"`
}

// webRunsResult is the top-level shape, matching the REST
// /actions/runs response so a downstream caller cannot tell which budget
// tier served the read.
type webRunsResult struct {
	TotalCount   int              `json:"total_count"`
	WorkflowRuns []webWorkflowRun `json:"workflow_runs"`
}

// readActionsRunsViaWeb fetches and parses the Actions page for owner/repo
// and returns the run list marshalled as the REST /actions/runs JSON
// shape. cookie is optional (see ghWebCookieEnv).
func readActionsRunsViaWeb(ctx context.Context, owner, repo, cookie string) ([]byte, error) {
	client := &http.Client{Timeout: ghWebReadTimeout}
	body, err := fetchActionsRunsHTML(ctx, client, owner, repo, cookie)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()
	runs, err := parseActionsRunsHTML(body, owner, repo)
	if err != nil {
		return nil, err
	}
	result := webRunsResult{TotalCount: len(runs), WorkflowRuns: runs}
	return json.Marshal(result)
}

// fetchActionsRunsHTML GETs the Actions page. A 200 with an HTML body is
// the success path. A redirect to the login page (GitHub answers an
// unauthenticated request for a private repo with a 302 to /login, or a
// 404 to avoid leaking existence) is reported as an error so the caller
// falls through to the next tier.
func fetchActionsRunsHTML(ctx context.Context, client *http.Client, owner, repo, cookie string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/%s/%s/actions", webRunsBaseURL, owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("gh actions web: build request: %w", err)
	}
	// A plain desktop UA: GitHub serves the lighter mobile shell to some
	// UA strings, and the run rows we parse live in the desktop shell.
	req.Header.Set("User-Agent", "coily-gh-actions-web")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gh actions web: GET %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("gh actions web: GET %s: status %d (private repo without %s, or page moved)", url, resp.StatusCode, ghWebCookieEnv)
	}
	return resp.Body, nil
}

var (
	// runHrefRe matches the run anchor href and captures the run id.
	runHrefRe = regexp.MustCompile(`^/[^/]+/[^/]+/actions/runs/(\d+)$`)
	// commitHrefRe matches a commit anchor and captures the full sha.
	commitHrefRe = regexp.MustCompile(`^/[^/]+/[^/]+/commit/([0-9a-f]{7,40})$`)
	// checkSuiteIDRe pulls the numeric id out of the row's element id.
	checkSuiteIDRe = regexp.MustCompile(`^check_suite_(\d+)$`)
	// ariaRe decomposes the run anchor aria-label:
	//   "<status phrase>:  Run <n> of <workflow>. <title>"
	// The title is optional - some runs render the phrase only.
	ariaRe = regexp.MustCompile(`(?s)^(.*?):\s+Run (\d+) of (.+?)\.(?:\s+(.*))?$`)
)

// parseActionsRunsHTML walks the parsed Actions page and extracts one
// webWorkflowRun per check-suite row. owner/repo are needed to build
// html_url. An unparseable row (no run id) is skipped, not fatal.
func parseActionsRunsHTML(r io.Reader, owner, repo string) ([]webWorkflowRun, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("gh actions web: parse HTML: %w", err)
	}
	var runs []webWorkflowRun
	for _, row := range findRunRows(doc) {
		if run, ok := parseRunRow(row, owner, repo); ok {
			runs = append(runs, run)
		}
	}
	return runs, nil
}

// findRunRows returns every element node whose id is "check_suite_<n>" -
// the container GitHub renders for one workflow run.
func findRunRows(root *html.Node) []*html.Node {
	var rows []*html.Node
	walkHTML(root, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		if checkSuiteIDRe.MatchString(nodeAttr(n, "id")) {
			rows = append(rows, n)
		}
	})
	return rows
}

// parseRunRow extracts a single run from one check-suite row. ok is false
// when the row carries no run anchor (the field everything else hangs
// off), so the caller skips it.
func parseRunRow(row *html.Node, owner, repo string) (webWorkflowRun, bool) {
	var run webWorkflowRun

	if m := checkSuiteIDRe.FindStringSubmatch(nodeAttr(row, "id")); m != nil {
		run.CheckSuiteID = atoi64(m[1])
	}

	var rowText strings.Builder
	walkHTML(row, func(n *html.Node) {
		switch {
		case n.Type == html.TextNode:
			rowText.WriteString(n.Data)
		case n.Type != html.ElementNode:
			return
		case n.Data == "a":
			classifyAnchor(n, &run)
		case n.Data == "relative-time" && run.CreatedAt == "":
			run.CreatedAt = nodeAttr(n, "datetime")
		}
	})

	if run.ID == 0 {
		return webWorkflowRun{}, false
	}
	run.HTMLURL = fmt.Sprintf("%s/%s/%s/actions/runs/%d", webRunsBaseURL, owner, repo, run.ID)
	run.Event = eventFromRowText(rowText.String())
	return run, true
}

// classifyAnchor inspects one <a> inside a run row and copies whatever
// field it carries into run. The run anchor (the one with an
// actions/runs/<id> href) also carries the aria-label that yields run
// number, workflow name, display title, status, and conclusion.
func classifyAnchor(n *html.Node, run *webWorkflowRun) {
	href := nodeAttr(n, "href")
	switch {
	case runHrefRe.MatchString(href):
		m := runHrefRe.FindStringSubmatch(href)
		run.ID = atoi64(m[1])
		applyAriaLabel(nodeAttr(n, "aria-label"), run)
	case commitHrefRe.MatchString(href) && run.HeadSHA == "":
		run.HeadSHA = commitHrefRe.FindStringSubmatch(href)[1]
	case strings.Contains(nodeAttr(n, "class"), "branch-name") && run.HeadBranch == "":
		run.HeadBranch = strings.TrimSpace(nodeAttr(n, "title"))
	case nodeAttr(n, "data-hovercard-type") == "user" && run.Actor == nil:
		if login := strings.TrimSpace(nodeText(n)); login != "" {
			run.Actor = &actor{Login: login}
		}
	}
}

// applyAriaLabel decomposes the run anchor's aria-label. On a shape it
// does not recognize it leaves run untouched rather than guessing.
func applyAriaLabel(label string, run *webWorkflowRun) {
	label = strings.TrimSpace(label)
	if label == "" {
		return
	}
	m := ariaRe.FindStringSubmatch(label)
	if m == nil {
		// No "Run <n> of <wf>" structure: still try the status phrase.
		run.Status, run.Conclusion = classifyStatusPhrase(label)
		return
	}
	run.Status, run.Conclusion = classifyStatusPhrase(m[1])
	run.RunNumber = atoi(m[2])
	run.Name = strings.TrimSpace(m[3])
	run.DisplayTitle = strings.TrimSpace(m[4])
}

// classifyStatusPhrase maps the human status phrase GitHub renders into
// the REST (status, conclusion) pair. A completed run carries a
// conclusion; an in-progress or queued run carries an empty conclusion,
// matching the REST API's null. An unrecognized phrase yields a
// best-effort status with no conclusion.
func classifyStatusPhrase(phrase string) (status, conclusion string) {
	p := strings.ToLower(strings.TrimSpace(phrase))
	switch {
	case strings.Contains(p, "success"):
		return "completed", "success"
	case strings.Contains(p, "fail"):
		// "completed with failures", "failed", "startup failure".
		return "completed", "failure"
	case strings.Contains(p, "cancel"):
		return "completed", "cancelled"
	case strings.Contains(p, "skip"):
		return "completed", "skipped"
	case strings.Contains(p, "timed out"):
		return "completed", "timed_out"
	case strings.Contains(p, "neutral"):
		return "completed", "neutral"
	case strings.Contains(p, "action required"):
		return "completed", "action_required"
	case strings.Contains(p, "in progress"), strings.Contains(p, "running"):
		return "in_progress", ""
	case strings.Contains(p, "queued"), strings.Contains(p, "waiting"), strings.Contains(p, "pending"):
		return "queued", ""
	default:
		return p, ""
	}
}

// eventFromRowText recovers the triggering event from the row's prose
// ("... pushed by ...", "... pull request ..."). The REST API reports a
// richer enum; this covers the two events that dominate CI rows and
// leaves the rest empty rather than mislabeling.
func eventFromRowText(text string) string {
	t := strings.ToLower(text)
	switch {
	case strings.Contains(t, "pull request"):
		return "pull_request"
	case strings.Contains(t, "pushed"):
		return "push"
	case strings.Contains(t, "scheduled"):
		return "schedule"
	default:
		return ""
	}
}

// walkHTML invokes fn on n and every descendant, pre-order.
func walkHTML(n *html.Node, fn func(*html.Node)) {
	fn(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkHTML(c, fn)
	}
}

// nodeAttr returns the value of attribute key on n, or "".
func nodeAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// nodeText returns the concatenated text content of n.
func nodeText(n *html.Node) string {
	var b strings.Builder
	walkHTML(n, func(c *html.Node) {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
	})
	return b.String()
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func atoi64(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

// ghWebReadTimeout caps a single Actions-page fetch. Generous enough for
// a cold github.com response, short enough that a hung tier yields to the
// next one promptly.
const ghWebReadTimeout = 15 * time.Second
