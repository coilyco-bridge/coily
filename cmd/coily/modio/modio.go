// Package modio is a thin wrapper over mod.io's v1 read API for the Eco
// game (game_id=6). The API key is fetched from AWS SSM at /modio/api-key
// each invocation - no caching, no env-var fallback - so the audit log
// never sees the secret and there is no on-disk credential surface beyond
// what AWS already manages.
//
// Verbs follow the mirror-verbatim style: each leaf maps 1:1 to a mod.io
// REST endpoint, JSON streamed to stdout for jq pipelines. Mutating
// endpoints (subscribe, rate, etc) need OAuth and are out of scope for
// this wrapper - if you need them, do it manually.
package modio

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

	"forgejo.coilysiren.me/coilyco-bridge/coily/cmd/coily/restfmt"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/cli/shell"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/cli/verb"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/pkg/audit"
	"github.com/urfave/cli/v3"
)

const (
	// DefaultGameID is mod.io's Eco game id. mod.io game ids are stable
	// per-game so baking this in is safe; --game-id overrides for any
	// other game's catalog.
	DefaultGameID = 6
	// SSMParamAPIKey is the SSM Parameter Store path that holds the
	// read-only mod.io API key. Per the canonical inventory in
	// agentic-os-kai/AGENTS.md.
	SSMParamAPIKey = "/modio/api-key" //nolint:gosec // SSM path, not a credential
	apiBase        = "https://api.mod.io/v1"
	httpTimeout    = 30 * time.Second
)

// Command returns the cli.Command tree for `coily ops modio`. The strict
// shell.Runner is reused for the SSM fetch so the api-key call goes
// through the same pinned-tool path as everything else.
func Command(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:  "modio",
		Usage: "Wrap the mod.io v1 REST API for Eco (game_id=6 by default).",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "game-id",
				Usage: "mod.io game id to scope every call to",
				Value: DefaultGameID,
			},
			// --query/--output mirror the generated REST surfaces
			// (coilyco-bridge/coily#46). urfave/cli v3 flags propagate to
			// subcommands by default, so every leaf inherits the restfmt
			// projection rail. No collision here - modio's API params are
			// limit/offset/game-id, so projection stays on --query.
			&cli.StringFlag{Name: "query", Usage: "JMESPath projection applied to the JSON response (coily#46)"},
			&cli.StringFlag{Name: "output", Usage: "output format: json (default), yaml, text (coily#46)"},
		},
		Commands: []*cli.Command{
			modsCommand(r, w),
		},
	}
}

func modsCommand(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:  "mods",
		Usage: "Mods endpoints under /games/{game-id}/mods.",
		Commands: []*cli.Command{
			modsList(r, w),
			modsGet(r, w),
			modsFiles(r, w),
			modsComments(r, w),
		},
	}
}

func modsList(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "GET /games/{game-id}/mods",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "limit", Usage: "max results per page", Value: 100},
			&cli.IntFlag{Name: "offset", Usage: "page offset"},
		},
		Action: verb.Wrap(
			verb.Spec{
				Name: "ops.modio.mods.list",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--game-id": fmt.Sprint(gameID(c)),
						"--limit":   fmt.Sprint(c.Int("limit")),
						"--offset":  fmt.Sprint(c.Int("offset")),
					}, nil
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					q := url.Values{}
					q.Set("_limit", fmt.Sprint(c.Int("limit")))
					if c.IsSet("offset") {
						q.Set("_offset", fmt.Sprint(c.Int("offset")))
					}
					return doGetAndStream(ctx, r, c,
						fmt.Sprintf("/games/%d/mods", gameID(c)), q)
				},
			},
			w,
		),
	}
}

func modsGet(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "GET /games/{game-id}/mods/{mod-id}",
		ArgsUsage: "<mod-id>",
		Action: verb.Wrap(
			verb.Spec{
				Name: "ops.modio.mods.get",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--game-id": fmt.Sprint(gameID(c))}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					modID, err := requireModID(c)
					if err != nil {
						return err
					}
					return doGetAndStream(ctx, r, c,
						fmt.Sprintf("/games/%d/mods/%s", gameID(c), modID), nil)
				},
			},
			w,
		),
	}
}

func modsFiles(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:      "files",
		Usage:     "GET /games/{game-id}/mods/{mod-id}/files",
		ArgsUsage: "<mod-id>",
		Action: verb.Wrap(
			verb.Spec{
				Name: "ops.modio.mods.files",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--game-id": fmt.Sprint(gameID(c))}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					modID, err := requireModID(c)
					if err != nil {
						return err
					}
					return doGetAndStream(ctx, r, c,
						fmt.Sprintf("/games/%d/mods/%s/files", gameID(c), modID), nil)
				},
			},
			w,
		),
	}
}

func modsComments(r *shell.Runner, w *audit.Writer) *cli.Command {
	return &cli.Command{
		Name:      "comments",
		Usage:     "GET /games/{game-id}/mods/{mod-id}/comments",
		ArgsUsage: "<mod-id>",
		Action: verb.Wrap(
			verb.Spec{
				Name: "ops.modio.mods.comments",
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--game-id": fmt.Sprint(gameID(c))}, c.Args().Slice()
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					modID, err := requireModID(c)
					if err != nil {
						return err
					}
					return doGetAndStream(ctx, r, c,
						fmt.Sprintf("/games/%d/mods/%s/comments", gameID(c), modID), nil)
				},
			},
			w,
		),
	}
}

// gameID resolves --game-id from the leaf or, if unset, walks up to the
// root `coily ops modio` command. Default 6.
func gameID(c *cli.Command) int {
	if g := c.Int("game-id"); g != 0 {
		return g
	}
	return DefaultGameID
}

func requireModID(c *cli.Command) (string, error) {
	if c.Args().Len() != 1 {
		return "", fmt.Errorf("modio: need exactly one <mod-id>, got %d", c.Args().Len())
	}
	id := c.Args().First()
	for _, ch := range id {
		if ch < '0' || ch > '9' {
			return "", fmt.Errorf("modio: <mod-id> must be numeric, got %q", id)
		}
	}
	return id, nil
}

// doGetAndStream fetches the mod.io endpoint with the SSM-resolved
// api_key as a query parameter, then streams the response body to stdout.
// 4xx/5xx surface as errors with the response body included so callers
// see the structured mod.io error envelope. When --query/--output are set
// the response routes through restfmt (coilyco-bridge/coily#46); otherwise
// the output is the byte-for-byte pretty-print as before.
func doGetAndStream(ctx context.Context, r *shell.Runner, c *cli.Command, path string, q url.Values) error {
	apiKey, err := fetchAPIKey(ctx, r)
	if err != nil {
		return err
	}
	if q == nil {
		q = url.Values{}
	}
	q.Set("api_key", apiKey)

	target := apiBase + path + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("modio: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("modio: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("modio: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("modio: read response: %w", err)
	}
	return streamResponse(c, raw)
}

// streamResponse writes the mod.io response to stdout. When --query/--output
// are set it routes through restfmt (coilyco-bridge/coily#46); otherwise it
// pretty-prints, byte-for-byte as before so jq pipelines stay practical.
func streamResponse(c *cli.Command, raw []byte) error {
	outFmt, outSet, err := restfmt.ParseOutput(c.String("output"))
	if err != nil {
		return err
	}
	if projection := c.String("query"); restfmt.NeedsRender(projection, outSet) {
		if !outSet {
			outFmt = restfmt.OutputJSON
		}
		rendered, rErr := restfmt.Render(projection, outFmt, raw)
		if rErr != nil {
			return rErr
		}
		_, err = os.Stdout.Write(rendered)
		return err
	}

	var pretty json.RawMessage
	if err := json.Unmarshal(raw, &pretty); err != nil {
		return fmt.Errorf("modio: decode response: %w", err)
	}
	out, err := json.MarshalIndent(pretty, "", "  ")
	if err != nil {
		return fmt.Errorf("modio: re-encode response: %w", err)
	}
	if _, err := fmt.Fprintln(os.Stdout, string(out)); err != nil {
		return err
	}
	return nil
}

// fetchAPIKey calls aws ssm get-parameter through the strict shell.Runner
// (the same path every other coily verb uses). Trims trailing newline so
// the value can be embedded directly in a URL.
func fetchAPIKey(ctx context.Context, r *shell.Runner) (string, error) {
	out, err := r.Capture(ctx, "aws",
		"ssm", "get-parameter",
		"--name", SSMParamAPIKey,
		"--with-decryption",
		"--query", "Parameter.Value",
		"--output", "text",
	)
	if err != nil {
		return "", fmt.Errorf("modio: fetch %s: %w", SSMParamAPIKey, err)
	}
	key := strings.TrimSpace(string(out))
	if key == "" {
		return "", fmt.Errorf("modio: %s resolved to empty value", SSMParamAPIKey)
	}
	return key, nil
}
