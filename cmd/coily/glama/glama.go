// Package glama wraps the Glama MCP v1 REST API
// (https://glama.ai/api/mcp/openapi.json). The directory endpoints
// (/v1/servers, /v1/attributes) are unauthenticated. /v1/instances
// requires a bearer token, fetched on demand from AWS SSM at
// /glama/api-key - no caching, no env-var fallback - so the audit log
// never sees the secret and there is no on-disk credential surface
// beyond what AWS already manages.
//
// Verbs follow the mirror-verbatim style: each leaf maps 1:1 to a Glama
// REST endpoint, JSON streamed to stdout for jq pipelines. The
// deprecated /v1/servers/{id} endpoint is intentionally skipped; use
// `servers get <namespace> <slug>` instead.
package glama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/shell"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
)

const (
	// SSMParamAPIKey is the SSM Parameter Store path that holds the
	// Glama bearer token. Only required for /v1/instances.
	SSMParamAPIKey = "/glama/api-key" //nolint:gosec // SSM path, not a credential
	apiBase        = "https://glama.ai/api/mcp"
	httpTimeout    = 30 * time.Second
)

// Command returns the cli.Command tree for `coily glama`.
func Command(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:  "glama",
		Usage: "Wrap the Glama MCP v1 REST API (https://glama.ai/api/mcp).",
		Commands: []*cli.Command{
			serversCommand(r, w),
			attributesCmd(r, w),
			instancesCmd(r, w),
			telemetryCommand(r, w),
		},
	}
}

func serversCommand(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:  "servers",
		Usage: "MCP server directory endpoints under /v1/servers.",
		Commands: []*cli.Command{
			serversList(r, w),
			serversGet(r, w),
		},
	}
}

func serversList(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "GET /v1/servers - paginated directory listing.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "query", Usage: "free-text search query"},
			&cli.IntFlag{Name: "first", Usage: "page size"},
			&cli.StringFlag{Name: "after", Usage: "opaque cursor from a prior page"},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name: "glama.servers.list",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					m := map[string]string{}
					if c.IsSet("query") {
						m["--query"] = c.String("query")
					}
					if c.IsSet("first") {
						m["--first"] = fmt.Sprint(c.Int("first"))
					}
					if c.IsSet("after") {
						m["--after"] = c.String("after")
					}
					return m, nil
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					q := url.Values{}
					if c.IsSet("query") {
						q.Set("query", c.String("query"))
					}
					if c.IsSet("first") {
						q.Set("first", fmt.Sprint(c.Int("first")))
					}
					if c.IsSet("after") {
						q.Set("after", c.String("after"))
					}
					return doGetAndStream(ctx, r, "/v1/servers", q, false)
				},
			},
			w,
		),
	}
}

func serversGet(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "GET /v1/servers/{namespace}/{slug}",
		ArgsUsage: "<namespace> <slug>",
		Action: verb.Wrap(
			verb.Spec{
				Name: "glama.servers.get",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return nil, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					ns, slug, err := requireNamespaceSlug(c)
					if err != nil {
						return err
					}
					return doGetAndStream(ctx, r,
						fmt.Sprintf("/v1/servers/%s/%s",
							url.PathEscape(ns), url.PathEscape(slug)),
						nil, false)
				},
			},
			w,
		),
	}
}

func attributesCmd(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:  "attributes",
		Usage: "GET /v1/attributes - the curated attribute vocabulary.",
		Action: verb.Wrap(
			verb.Spec{
				Name: "glama.attributes",
				ArgsFunc: func(_ *cli.Command) (map[string]string, []string) {
					return nil, nil
				},
				Action: func(ctx context.Context, _ *cli.Command) error {
					return doGetAndStream(ctx, r, "/v1/attributes", nil, false)
				},
			},
			w,
		),
	}
}

func instancesCmd(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:  "instances",
		Usage: "GET /v1/instances - hosted MCP server instances (bearer auth).",
		Action: verb.Wrap(
			verb.Spec{
				Name: "glama.instances",
				ArgsFunc: func(_ *cli.Command) (map[string]string, []string) {
					return nil, nil
				},
				Action: func(ctx context.Context, _ *cli.Command) error {
					return doGetAndStream(ctx, r, "/v1/instances", nil, true)
				},
			},
			w,
		),
	}
}

func telemetryCommand(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:  "telemetry",
		Usage: "Telemetry endpoints under /v1/telemetry.",
		Commands: []*cli.Command{
			telemetryUsage(r, w),
		},
	}
}

func telemetryUsage(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:  "usage",
		Usage: "POST /v1/telemetry/usage - send tool-usage data.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "body",
				Usage: "JSON body, '@<path>' to read a file, or '-' for stdin",
			},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name: "glama.telemetry.usage",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					m := map[string]string{}
					if c.IsSet("body") {
						m["--body"] = c.String("body")
					}
					return m, nil
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					body, err := readBody(c.String("body"))
					if err != nil {
						return err
					}
					return doPostAndStream(ctx, r, "/v1/telemetry/usage", body, false)
				},
			},
			w,
		),
	}
}

func requireNamespaceSlug(c *cli.Command) (string, string, error) {
	args := c.Args().Slice()
	if len(args) != 2 {
		return "", "", fmt.Errorf("glama: need <namespace> <slug>, got %d args", len(args))
	}
	if args[0] == "" || args[1] == "" {
		return "", "", fmt.Errorf("glama: <namespace> and <slug> must be non-empty")
	}
	return args[0], args[1], nil
}

// readBody resolves the --body flag: literal JSON, '@path' file read, or
// '-' for stdin. Empty input returns an empty body for endpoints that
// accept it.
func readBody(spec string) ([]byte, error) {
	switch {
	case spec == "":
		return nil, nil
	case spec == "-":
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("glama: read stdin: %w", err)
		}
		return b, nil
	case strings.HasPrefix(spec, "@"):
		b, err := os.ReadFile(spec[1:])
		if err != nil {
			return nil, fmt.Errorf("glama: read %s: %w", spec[1:], err)
		}
		return b, nil
	default:
		return []byte(spec), nil
	}
}

// doGetAndStream fetches a Glama endpoint and streams the JSON response
// to stdout pretty-printed. authRequired triggers an SSM fetch for the
// bearer token; unauthenticated endpoints skip it entirely so directory
// browsing works without AWS credentials.
func doGetAndStream(ctx context.Context, r *shell.Runner, path string, q url.Values, authRequired bool) error {
	target := apiBase + path
	if len(q) > 0 {
		target += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("glama: build request: %w", err)
	}
	return doRequestAndStream(ctx, r, req, authRequired)
}

func doPostAndStream(ctx context.Context, r *shell.Runner, path string, body []byte, authRequired bool) error {
	var rdr io.Reader
	if len(body) > 0 {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+path, rdr)
	if err != nil {
		return fmt.Errorf("glama: build request: %w", err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	return doRequestAndStream(ctx, r, req, authRequired)
}

func doRequestAndStream(ctx context.Context, r *shell.Runner, req *http.Request, authRequired bool) error {
	req.Header.Set("Accept", "application/json")
	if authRequired {
		apiKey, err := fetchAPIKey(ctx, r)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("glama: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("glama: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("glama: read response: %w", err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	var pretty json.RawMessage
	if err := json.Unmarshal(raw, &pretty); err != nil {
		// Non-JSON body: emit verbatim so the caller sees what came back.
		_, _ = os.Stdout.Write(raw)
		return nil
	}
	out, err := json.MarshalIndent(pretty, "", "  ")
	if err != nil {
		return fmt.Errorf("glama: re-encode response: %w", err)
	}
	if _, err := fmt.Fprintln(os.Stdout, string(out)); err != nil {
		return err
	}
	return nil
}

// fetchAPIKey calls aws ssm get-parameter through the strict shell.Runner.
func fetchAPIKey(ctx context.Context, r *shell.Runner) (string, error) {
	out, err := r.Capture(ctx, "aws",
		"ssm", "get-parameter",
		"--name", SSMParamAPIKey,
		"--with-decryption",
		"--query", "Parameter.Value",
		"--output", "text",
	)
	if err != nil {
		return "", fmt.Errorf("glama: fetch %s: %w", SSMParamAPIKey, err)
	}
	key := strings.TrimSpace(string(out))
	if key == "" {
		return "", fmt.Errorf("glama: %s resolved to empty value", SSMParamAPIKey)
	}
	return key, nil
}
