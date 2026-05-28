package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// forgejoAPIHTTPTimeout caps each forgejo API call so a stalled DNS or
// network hop doesn't hang the audit row indefinitely.
const forgejoAPIHTTPTimeout = 20 * time.Second

// forgejoIssueListDefaultLimit is the default cap on `issue list` results
// when the operator doesn't pass --limit. 30 matches forgejo's own page
// size default.
const forgejoIssueListDefaultLimit = 30

// forgejoIssueCommand is the `issue` subtree under `coily ops forgejo`.
// Hits the forgejo HTTP API directly (token from SSM) rather than going
// through `k3s kubectl exec`. Closes coilysiren/coily#69 (create) and
// coilysiren/coily#70 (read + state verbs).
func (r *Runner) forgejoIssueCommand() *cli.Command {
	return &cli.Command{
		Name:  "issue",
		Usage: "Forgejo issue API verbs (HTTP, not kubectl exec).",
		Commands: []*cli.Command{
			r.forgejoIssueCreateCommand(),
			r.forgejoIssueListCommand(),
			r.forgejoIssueViewCommand(),
			r.forgejoIssueEditCommand(),
			r.forgejoIssueCommentCommand(),
			r.forgejoIssueCloseCommand(),
			r.forgejoIssueReopenCommand(),
			r.forgejoIssueDeleteCommand(),
		},
	}
}

func (r *Runner) forgejoIssueCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a forgejo issue via the API.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.StringFlag{Name: "title", Usage: "issue title", Required: true},
			&cli.StringFlag{Name: "body-file", Usage: "path to the issue body markdown", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:       "ops.forgejo.issue.create",
				SkipPolicy: true,
				OnComplete: stampPolicySkipped,
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo issue create: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					title := strings.TrimSpace(c.String("title"))
					if title == "" {
						return fmt.Errorf("ops forgejo issue create: --title is empty")
					}
					body, err := readForgejoBodyFile(c.String("body-file"), "ops forgejo issue create")
					if err != nil {
						return err
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/issues", owner, repo)
					payload, err := json.Marshal(forgejoIssueCreateBody{Title: title, Body: body})
					if err != nil {
						return fmt.Errorf("ops forgejo issue create: marshal payload: %w", err)
					}
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo issue create", http.MethodPost, path, payload, http.StatusCreated)
					if err != nil {
						return err
					}
					return printForgejoIssueSummary(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoIssueListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List forgejo issues in a repo.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.StringFlag{Name: "state", Usage: "open | closed | all", Value: "open"},
			&cli.IntFlag{Name: "limit", Usage: "max results to return", Value: forgejoIssueListDefaultLimit},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.issue.list",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":  c.String("repo"),
						"--state": c.String("state"),
						"--limit": strconv.Itoa(c.Int("limit")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo issue list: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					state := c.String("state")
					if state != "open" && state != "closed" && state != "all" {
						return fmt.Errorf("ops forgejo issue list: --state must be open | closed | all, got %q", state)
					}
					limit := c.Int("limit")
					if limit <= 0 || limit > 50 {
						return fmt.Errorf("ops forgejo issue list: --limit must be 1..50, got %d", limit)
					}
					q := url.Values{}
					q.Set("state", state)
					q.Set("limit", strconv.Itoa(limit))
					q.Set("type", "issues") // exclude PRs; forgejo's /issues endpoint folds them in by default
					path := fmt.Sprintf("/api/v1/repos/%s/%s/issues?%s", owner, repo, q.Encode())
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo issue list", http.MethodGet, path, nil, http.StatusOK)
					if err != nil {
						return err
					}
					return printForgejoIssueList(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoIssueViewCommand() *cli.Command {
	return &cli.Command{
		Name:  "view",
		Usage: "View a single forgejo issue.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "number", Usage: "issue number", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.issue.view",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":   c.String("repo"),
						"--number": strconv.Itoa(c.Int("number")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo issue view: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					num, err := validateForgejoIssueNumber(c.Int("number"), "ops forgejo issue view")
					if err != nil {
						return err
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d", owner, repo, num)
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo issue view", http.MethodGet, path, nil, http.StatusOK)
					if err != nil {
						return err
					}
					return printForgejoIssueDetail(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoIssueEditCommand() *cli.Command {
	return &cli.Command{
		Name:  "edit",
		Usage: "Edit an existing forgejo issue's title and/or body.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "number", Usage: "issue number", Required: true},
			&cli.StringFlag{Name: "title", Usage: "new issue title (optional)"},
			&cli.StringFlag{Name: "body-file", Usage: "path to the new issue body markdown (optional)"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:       "ops.forgejo.issue.edit",
				SkipPolicy: true,
				OnComplete: stampPolicySkipped,
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo issue edit: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					num, err := validateForgejoIssueNumber(c.Int("number"), "ops forgejo issue edit")
					if err != nil {
						return err
					}
					title := strings.TrimSpace(c.String("title"))
					bodyPath := c.String("body-file")
					if title == "" && bodyPath == "" {
						return fmt.Errorf("ops forgejo issue edit: at least one of --title or --body-file is required")
					}
					patch := forgejoIssueEditBody{}
					if title != "" {
						patch.Title = &title
					}
					if bodyPath != "" {
						body, rerr := readForgejoBodyFile(bodyPath, "ops forgejo issue edit")
						if rerr != nil {
							return rerr
						}
						patch.Body = &body
					}
					payload, err := json.Marshal(patch)
					if err != nil {
						return fmt.Errorf("ops forgejo issue edit: marshal payload: %w", err)
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d", owner, repo, num)
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo issue edit", http.MethodPatch, path, payload, http.StatusCreated)
					if err != nil {
						return err
					}
					return printForgejoIssueSummary(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoIssueCommentCommand() *cli.Command {
	return &cli.Command{
		Name:  "comment",
		Usage: "Add a comment to a forgejo issue.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "number", Usage: "issue number", Required: true},
			&cli.StringFlag{Name: "body-file", Usage: "path to the comment markdown", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.issue.comment",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":      c.String("repo"),
						"--number":    strconv.Itoa(c.Int("number")),
						"--body-file": c.String("body-file"),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo issue comment: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					num, err := validateForgejoIssueNumber(c.Int("number"), "ops forgejo issue comment")
					if err != nil {
						return err
					}
					body, err := readForgejoBodyFile(c.String("body-file"), "ops forgejo issue comment")
					if err != nil {
						return err
					}
					payload, err := json.Marshal(forgejoIssueCommentBody{Body: body})
					if err != nil {
						return fmt.Errorf("ops forgejo issue comment: marshal payload: %w", err)
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/comments", owner, repo, num)
					respBody, err := r.forgejoAPIDo(ctx, "ops forgejo issue comment", http.MethodPost, path, payload, http.StatusCreated)
					if err != nil {
						return err
					}
					return printForgejoCommentSummary(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoIssueCloseCommand() *cli.Command {
	return r.forgejoIssueStateCommand("close", "closed")
}

func (r *Runner) forgejoIssueReopenCommand() *cli.Command {
	return r.forgejoIssueStateCommand("reopen", "open")
}

// forgejoIssueStateCommand factors the close/reopen pair which differ
// only in the state value PATCHed. Both target the same endpoint and the
// same audit shape; mirrors how gh's `issue close` and `issue reopen`
// collapse onto a single state mutation.
func (r *Runner) forgejoIssueStateCommand(name, state string) *cli.Command {
	verbName := "ops.forgejo.issue." + name
	errPrefix := "ops forgejo issue " + name
	return &cli.Command{
		Name:  name,
		Usage: fmt.Sprintf("Set a forgejo issue's state to %s.", state),
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "number", Usage: "issue number", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: verbName,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":   c.String("repo"),
						"--number": strconv.Itoa(c.Int("number")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("%s: takes no positional args, got %d", errPrefix, c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					num, err := validateForgejoIssueNumber(c.Int("number"), errPrefix)
					if err != nil {
						return err
					}
					payload, err := json.Marshal(forgejoIssueStateBody{State: state})
					if err != nil {
						return fmt.Errorf("%s: marshal payload: %w", errPrefix, err)
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d", owner, repo, num)
					respBody, err := r.forgejoAPIDo(ctx, errPrefix, http.MethodPatch, path, payload, http.StatusCreated)
					if err != nil {
						return err
					}
					return printForgejoIssueSummary(respBody)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) forgejoIssueDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a forgejo issue (forgejo requires site-admin).",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "repo", Usage: "target repo as owner/name", Required: true},
			&cli.IntFlag{Name: "number", Usage: "issue number", Required: true},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "ops.forgejo.issue.delete",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":   c.String("repo"),
						"--number": strconv.Itoa(c.Int("number")),
					}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("ops forgejo issue delete: takes no positional args, got %d", c.Args().Len())
					}
					owner, repo, err := parseForgejoRepoSlug(c.String("repo"))
					if err != nil {
						return err
					}
					num, err := validateForgejoIssueNumber(c.Int("number"), "ops forgejo issue delete")
					if err != nil {
						return err
					}
					path := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d", owner, repo, num)
					if _, err := r.forgejoAPIDo(ctx, "ops forgejo issue delete", http.MethodDelete, path, nil, http.StatusNoContent); err != nil {
						return err
					}
					fmt.Printf("deleted %s/%s#%d\n", owner, repo, num)
					return nil
				},
			},
			r.Audit,
		),
	}
}

// forgejoAPIDo is the shared HTTP path for all forgejo API verbs. Fetches
// the bearer token from SSM, builds the request, enforces the expected
// HTTP status, and returns the response body. payload may be nil for
// GET/DELETE. wantStatus is the single status code the verb treats as
// success (forgejo's API is consistent enough that this is fine; if a
// future verb needs 200/201 either-or, widen the type).
func (r *Runner) forgejoAPIDo(ctx context.Context, prefix, method, path string, payload []byte, wantStatus int) ([]byte, error) {
	base := strings.TrimSuffix(r.Cfg.Forgejo.BaseURL, "/")
	if base == "" {
		return nil, fmt.Errorf("%s: forgejo.base_url not set in config", prefix)
	}
	ssmPath := r.Cfg.Forgejo.SSMTokenPath
	if ssmPath == "" {
		return nil, fmt.Errorf("%s: forgejo.ssm_token_path not set in config", prefix)
	}
	token, err := r.forgejoAPIToken(ctx, prefix, ssmPath)
	if err != nil {
		return nil, err
	}
	target := base + path

	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", prefix, err)
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: forgejoAPIHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %s %s: %w", prefix, method, target, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		return nil, fmt.Errorf("%s: %s %s returned HTTP %d: %s", prefix, method, target, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return respBody, nil
}

// forgejoAPIToken resolves the bearer token from AWS SSM. Mirrors
// channelToken in channel.go — same r.Runner.Capture path so the aws CLI
// is found via $PATH and credentials come from the operator's profile.
func (r *Runner) forgejoAPIToken(ctx context.Context, prefix, ssmPath string) (string, error) {
	out, err := r.Runner.Capture(ctx, "aws",
		"ssm", "get-parameter",
		"--name", ssmPath,
		"--with-decryption",
		"--query", "Parameter.Value",
		"--output", "text",
	)
	if err != nil {
		return "", fmt.Errorf("%s: fetch %s: %w", prefix, ssmPath, err)
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return "", fmt.Errorf("%s: %s resolved to empty value", prefix, ssmPath)
	}
	return v, nil
}

// readForgejoBodyFile slurps a markdown body from disk and validates it
// non-empty. All --body-file flags route through here so the error
// message shape is consistent.
func readForgejoBodyFile(path, prefix string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s: read --body-file %s: %w", prefix, path, err)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return "", fmt.Errorf("%s: --body-file %s is empty", prefix, path)
	}
	return string(body), nil
}

// validateForgejoIssueNumber rejects non-positive issue numbers. Forgejo
// rejects them too, but failing here keeps the audit row clean and saves
// a round trip.
func validateForgejoIssueNumber(n int, prefix string) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("%s: --number must be >= 1, got %d", prefix, n)
	}
	return n, nil
}

// forgejoIssueCreateBody / forgejoIssueEditBody / forgejoIssueStateBody /
// forgejoIssueCommentBody are the POST/PATCH payloads. Edit fields are
// pointers so unset fields stay out of the marshalled JSON (forgejo's
// PATCH semantics treat a missing key as "do not change"; an empty
// string would clear it).
type forgejoIssueCreateBody struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type forgejoIssueEditBody struct {
	Title *string `json:"title,omitempty"`
	Body  *string `json:"body,omitempty"`
}

type forgejoIssueStateBody struct {
	State string `json:"state"`
}

type forgejoIssueCommentBody struct {
	Body string `json:"body"`
}

// forgejoIssueSummary is the subset of the forgejo issue response we
// surface to stdout for create/edit/close/reopen. Forgejo returns far
// more; we ignore everything else.
type forgejoIssueSummary struct {
	Number  int    `json:"number"`
	State   string `json:"state"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
}

// forgejoCommentSummary is the subset of the comment response we
// surface to stdout for `issue comment`.
type forgejoCommentSummary struct {
	ID      int    `json:"id"`
	HTMLURL string `json:"html_url"`
}

func printForgejoIssueSummary(respBody []byte) error {
	var out forgejoIssueSummary
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo issue response: %w", err)
	}
	fmt.Printf("%s #%d [%s] %s\n", out.HTMLURL, out.Number, out.State, out.Title)
	return nil
}

func printForgejoIssueDetail(respBody []byte) error {
	var out struct {
		forgejoIssueSummary
		Body string `json:"body"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo issue response: %w", err)
	}
	fmt.Printf("%s #%d [%s] %s\n", out.HTMLURL, out.Number, out.State, out.Title)
	if strings.TrimSpace(out.Body) != "" {
		fmt.Printf("\n%s\n", out.Body)
	}
	return nil
}

func printForgejoIssueList(respBody []byte) error {
	var out []forgejoIssueSummary
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo issue list: %w", err)
	}
	for _, it := range out {
		fmt.Printf("%s\t#%d\t[%s]\t%s\n", it.HTMLURL, it.Number, it.State, it.Title)
	}
	return nil
}

func printForgejoCommentSummary(respBody []byte) error {
	var out forgejoCommentSummary
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("decode forgejo comment response: %w", err)
	}
	fmt.Printf("%s comment #%d\n", out.HTMLURL, out.ID)
	return nil
}

// parseForgejoRepoSlug splits owner/repo and applies the same shape rules
// as forgejo's own URL routing: non-empty owner and repo, no slashes
// within either side, no leading dash on either side (argv-as-flag guard
// applied even though this never reaches a shell — defense in depth).
func parseForgejoRepoSlug(slug string) (string, string, error) {
	parts := strings.Split(slug, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("ops forgejo: --repo must be owner/name, got %q", slug)
	}
	owner, repo := parts[0], parts[1]
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("ops forgejo: --repo must be owner/name with non-empty parts, got %q", slug)
	}
	if strings.HasPrefix(owner, "-") || strings.HasPrefix(repo, "-") {
		return "", "", fmt.Errorf("ops forgejo: --repo parts must not start with '-', got %q", slug)
	}
	if err := validateForgejoSlugPart(owner); err != nil {
		return "", "", err
	}
	if err := validateForgejoSlugPart(repo); err != nil {
		return "", "", err
	}
	return owner, repo, nil
}

// validateForgejoSlugPart restricts owner/repo to the GitHub-compatible
// character set forgejo also accepts (alnum + - _ .). Forgejo does the
// real existence check; this rejects obvious shapes early.
func validateForgejoSlugPart(s string) error {
	if len(s) > 100 {
		return fmt.Errorf("ops forgejo: --repo part %q too long", s)
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
		default:
			return fmt.Errorf("ops forgejo: --repo part %q contains invalid character %q", s, r)
		}
	}
	return nil
}
