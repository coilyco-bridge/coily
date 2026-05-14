// Package skillsmp wraps the skillsmp.com v1 read API
// (https://skillsmp.com/api/v1/skills). The bearer token is fetched from
// AWS SSM at /skillsmp/api-key each invocation - no caching, no env-var
// fallback - so the audit log never sees the secret and there is no
// on-disk credential surface beyond what AWS already manages.
//
// Verbs follow the mirror-verbatim style: each leaf maps 1:1 to a
// skillsmp REST endpoint, JSON streamed to stdout for jq pipelines. Only
// GET endpoints are exposed (the API itself is read-only at v1).
package skillsmp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coilysiren/cli-guard/audit"
	"github.com/coilysiren/cli-guard/shell"
	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

const (
	// SSMParamAPIKey is the SSM Parameter Store path that holds the
	// skillsmp.com bearer token. Per the canonical inventory in
	// coilyco-ai/AGENTS.md.
	SSMParamAPIKey = "/skillsmp/api-key" //nolint:gosec // SSM path, not a credential
	apiBase        = "https://skillsmp.com/api/v1/skills"
	httpTimeout    = 30 * time.Second
)

// Command returns the cli.Command tree for `coily pkg skillsmp`.
func Command(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:  "skillsmp",
		Usage: "Wrap the skillsmp.com v1 read API for skill discovery.",
		Commands: []*cli.Command{
			searchCmd(r, w),
			aiSearchCmd(r, w),
		},
	}
}

func searchCmd(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:      "search",
		Usage:     "GET /search - keyword search across skills.",
		ArgsUsage: "<query>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "sort-by", Usage: "sort field (e.g. stars)", Value: "stars"},
			&cli.IntFlag{Name: "limit", Usage: "results per page"},
			&cli.IntFlag{Name: "page", Usage: "page number (1-indexed)"},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name: "skillsmp.search",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					args := map[string]string{"--sort-by": c.String("sort-by")}
					if c.IsSet("limit") {
						args["--limit"] = fmt.Sprint(c.Int("limit"))
					}
					if c.IsSet("page") {
						args["--page"] = fmt.Sprint(c.Int("page"))
					}
					return args, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					q, err := requireQuery(c)
					if err != nil {
						return err
					}
					vals := url.Values{}
					vals.Set("q", q)
					vals.Set("sortBy", c.String("sort-by"))
					if c.IsSet("limit") {
						vals.Set("limit", fmt.Sprint(c.Int("limit")))
					}
					if c.IsSet("page") {
						vals.Set("page", fmt.Sprint(c.Int("page")))
					}
					return doGetAndStream(ctx, r, "/search", vals)
				},
			},
			w,
		),
	}
}

func aiSearchCmd(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:      "ai-search",
		Usage:     "GET /ai-search - semantic search across skills.",
		ArgsUsage: "<query>",
		Action: verb.Wrap(
			verb.Spec{
				Name: "skillsmp.ai-search",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return nil, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					q, err := requireQuery(c)
					if err != nil {
						return err
					}
					vals := url.Values{}
					vals.Set("q", q)
					return doGetAndStream(ctx, r, "/ai-search", vals)
				},
			},
			w,
		),
	}
}

// requireQuery joins all positional args with a space so a multi-word
// natural-language query works without quoting (`coily pkg skillsmp ai-search
// parse edifact files`). Empty positional list is an error.
func requireQuery(c *cli.Command) (string, error) {
	args := c.Args().Slice()
	if len(args) == 0 {
		return "", fmt.Errorf("skillsmp: need <query>")
	}
	return strings.Join(args, " "), nil
}

// doGetAndStream fetches the skillsmp endpoint with the SSM-resolved
// bearer token, then streams the response body to stdout. 4xx/5xx
// surface as errors with the response body included.
func doGetAndStream(ctx context.Context, r *shell.Runner, path string, q url.Values) error {
	apiKey, err := fetchAPIKey(ctx, r)
	if err != nil {
		return err
	}
	target := apiBase + path
	if len(q) > 0 {
		target += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("skillsmp: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("skillsmp: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("skillsmp: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var pretty json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&pretty); err != nil {
		return fmt.Errorf("skillsmp: decode response: %w", err)
	}
	out, err := json.MarshalIndent(pretty, "", "  ")
	if err != nil {
		return fmt.Errorf("skillsmp: re-encode response: %w", err)
	}
	if _, err := fmt.Fprintln(os.Stdout, string(out)); err != nil {
		return err
	}
	return nil
}

// fetchAPIKey calls aws ssm get-parameter through the strict shell.Runner
// (the same path every other coily verb uses).
func fetchAPIKey(ctx context.Context, r *shell.Runner) (string, error) {
	out, err := r.Capture(ctx, "aws",
		"ssm", "get-parameter",
		"--name", SSMParamAPIKey,
		"--with-decryption",
		"--query", "Parameter.Value",
		"--output", "text",
	)
	if err != nil {
		return "", fmt.Errorf("skillsmp: fetch %s: %w", SSMParamAPIKey, err)
	}
	key := strings.TrimSpace(string(out))
	if key == "" {
		return "", fmt.Errorf("skillsmp: %s resolved to empty value", SSMParamAPIKey)
	}
	return key, nil
}
