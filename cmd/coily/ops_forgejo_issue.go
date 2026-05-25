package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// forgejoIssueHTTPTimeout caps the forgejo API call so a stalled DNS or
// network hop doesn't hang the audit row indefinitely.
const forgejoIssueHTTPTimeout = 20 * time.Second

// forgejoIssueCommand is the `issue` subtree under `coily ops forgejo`.
// Distinct mechanism from the rest of the file: hits the forgejo HTTP API
// directly (token from SSM) rather than going through `k3s kubectl exec`.
// Closes coilysiren/coily#69 — replaces the curl+SSM+temp-JSON dance
// agents had to do to file forgejo tickets.
func (r *Runner) forgejoIssueCommand() *cli.Command {
	return &cli.Command{
		Name:  "issue",
		Usage: "Forgejo issue API verbs (HTTP, not kubectl exec).",
		Commands: []*cli.Command{
			r.forgejoIssueCreateCommand(),
		},
	}
}

// forgejoIssueCreateCommand POSTs a new issue to the forgejo API.
// --body-file (rather than --body) is intentional: matches the agent
// workflow where the lockdown policy gate rejects shell metacharacters
// in positional/flag values, so multi-paragraph markdown has to come from
// a file anyway.
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
				Name: "ops.forgejo.issue.create",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--repo":      c.String("repo"),
						"--title":     c.String("title"),
						"--body-file": c.String("body-file"),
					}, c.Args().Slice()
				},
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
					bodyPath := c.String("body-file")
					body, err := os.ReadFile(bodyPath)
					if err != nil {
						return fmt.Errorf("ops forgejo issue create: read --body-file %s: %w", bodyPath, err)
					}
					return r.runForgejoIssueCreate(ctx, owner, repo, title, string(body))
				},
			},
			r.Audit,
		),
	}
}

// runForgejoIssueCreate fetches the API token from SSM, POSTs the issue,
// prints `<html_url> #<number>` on success. Token is fetched fresh per
// call — no on-disk cache here; this verb is low-volume by design.
func (r *Runner) runForgejoIssueCreate(ctx context.Context, owner, repo, title, body string) error {
	base := strings.TrimSuffix(r.Cfg.Forgejo.BaseURL, "/")
	if base == "" {
		return fmt.Errorf("ops forgejo issue create: forgejo.base_url not set in config")
	}
	ssmPath := r.Cfg.Forgejo.SSMTokenPath
	if ssmPath == "" {
		return fmt.Errorf("ops forgejo issue create: forgejo.ssm_token_path not set in config")
	}

	token, err := r.forgejoAPIToken(ctx, ssmPath)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(forgejoIssueCreateBody{Title: title, Body: body})
	if err != nil {
		return fmt.Errorf("ops forgejo issue create: marshal payload: %w", err)
	}

	target := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues", base, owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("ops forgejo issue create: build request: %w", err)
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: forgejoIssueHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ops forgejo issue create: POST %s: %w", target, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("ops forgejo issue create: %s returned HTTP %d: %s", target, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var out forgejoIssueCreateResp
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("ops forgejo issue create: decode response: %w", err)
	}
	fmt.Printf("%s #%d\n", out.HTMLURL, out.Number)
	return nil
}

// forgejoAPIToken resolves the bearer token from AWS SSM. Mirrors
// channelToken in channel.go — same r.Runner.Capture path so the aws CLI
// is found via $PATH and credentials come from the operator's profile.
func (r *Runner) forgejoAPIToken(ctx context.Context, ssmPath string) (string, error) {
	out, err := r.Runner.Capture(ctx, "aws",
		"ssm", "get-parameter",
		"--name", ssmPath,
		"--with-decryption",
		"--query", "Parameter.Value",
		"--output", "text",
	)
	if err != nil {
		return "", fmt.Errorf("ops forgejo issue create: fetch %s: %w", ssmPath, err)
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return "", fmt.Errorf("ops forgejo issue create: %s resolved to empty value", ssmPath)
	}
	return v, nil
}

// forgejoIssueCreateBody is the POST payload. Only the required fields;
// label/milestone/assignee can land in a follow-up if needed.
type forgejoIssueCreateBody struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// forgejoIssueCreateResp is the subset of the forgejo issue response we
// surface to stdout. Forgejo returns far more; we ignore everything else.
type forgejoIssueCreateResp struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
}

// parseForgejoRepoSlug splits owner/repo and applies the same shape rules
// as forgejo's own URL routing: non-empty owner and repo, no slashes
// within either side, no leading dash on either side (argv-as-flag guard
// applied even though this never reaches a shell — defense in depth).
func parseForgejoRepoSlug(slug string) (string, string, error) {
	parts := strings.Split(slug, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("ops forgejo issue create: --repo must be owner/name, got %q", slug)
	}
	owner, repo := parts[0], parts[1]
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("ops forgejo issue create: --repo must be owner/name with non-empty parts, got %q", slug)
	}
	if strings.HasPrefix(owner, "-") || strings.HasPrefix(repo, "-") {
		return "", "", fmt.Errorf("ops forgejo issue create: --repo parts must not start with '-', got %q", slug)
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
		return fmt.Errorf("ops forgejo issue create: --repo part %q too long", s)
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
		default:
			return fmt.Errorf("ops forgejo issue create: --repo part %q contains invalid character %q", s, r)
		}
	}
	return nil
}
