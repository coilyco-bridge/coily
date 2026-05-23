// channel.go wires `coily channel` - the operator CLI surface for Agent
// Channel coordination (coilysiren/coily#330). Each verb is a thin HTTP
// wrapper over the v2 Agent Channel protocol exposed by the reference
// deployment in coilysiren/backend. Token and base URL live behind the
// wrapper so the Claude Code classifier sees a single audited coily call
// instead of an SSM SecureString fetch chained into a raw curl.
//
// Protocol: https://github.com/coilysiren/otel-a2a-relay/blob/main/docs/channels-protocol.md
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
	"strings"
	"time"

	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

const channelHTTPTimeout = 60 * time.Second

func (r *Runner) channelCommand() *cli.Command {
	return &cli.Command{
		Name:  "channel",
		Usage: "Agent Channel coordination (create/post/read/state/spec/events/close).",
		Description: `channel wraps the v2 Agent Channel protocol exposed by the reference
deployment in coilysiren/backend. The bearer token is resolved from AWS
SSM at config channel.ssm_token_path (default /coilysiren/backend/
datastore-token); the base URL comes from config channel.base_url
(default http://api - the tailnet sidecar exposed by the reference
deployment).

Verbs:
  coily channel create --title <s>
  coily channel post   <id> --kind <k> --author <a> [--payload <file|->]
  coily channel read   <id> [--format json|yaml|markdown]
  coily channel state  <id>
  coily channel spec   <id>
  coily channel events <id> [--kind <k>] [--limit <n>]
  coily channel close  <id>

Protocol: https://github.com/coilysiren/otel-a2a-relay/blob/main/docs/channels-protocol.md`,
		Commands: []*cli.Command{
			r.channelCreateCommand(),
			r.channelPostCommand(),
			r.channelReadCommand(),
			r.channelStateCommand(),
			r.channelSpecCommand(),
			r.channelEventsCommand(),
			r.channelCloseCommand(),
		},
	}
}

func (r *Runner) channelCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a channel. Returns the fresh 4-char id and URL.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "title", Usage: "human-readable channel title", Required: true},
			&cli.StringFlag{Name: "created-by", Usage: "creator agent id (defaults to `coily agent-name`)"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "channel.create",
				SkipScope: true,
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 0 {
						return fmt.Errorf("channel create: takes no positional args, got %d", c.Args().Len())
					}
					body := map[string]any{"title": c.String("title")}
					if cb := c.String("created-by"); cb != "" {
						body["created_by"] = cb
					}
					raw, err := json.Marshal(body)
					if err != nil {
						return fmt.Errorf("channel create: marshal: %w", err)
					}
					return r.channelDo(ctx, "POST", "/agent-channel", nil, raw, "")
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) channelPostCommand() *cli.Command {
	return &cli.Command{
		Name:      "post",
		Usage:     "Append an event to a channel. Payload from --payload (literal | @file | -).",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "kind", Usage: "event kind (spec, state, status, comms, log, ...)", Required: true},
			&cli.StringFlag{Name: "author", Usage: "author agent id (defaults to `coily agent-name`)"},
			&cli.StringFlag{Name: "payload", Usage: "payload JSON: literal | @file | - for stdin"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name: "channel.post",
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := channelID(c.Name, c.Args().Slice())
					if err != nil {
						return err
					}
					payload, err := readChannelPayload(c.String("payload"))
					if err != nil {
						return err
					}
					body := map[string]any{
						"kind":    c.String("kind"),
						"payload": payload,
					}
					if a := c.String("author"); a != "" {
						body["author"] = a
					}
					raw, err := json.Marshal(body)
					if err != nil {
						return fmt.Errorf("channel post: marshal: %w", err)
					}
					return r.channelDo(ctx, "POST", "/agent-channel/"+url.PathEscape(id)+"/event", nil, raw, "")
				},
			},
			r.Audit,
		),
		SkipFlagParsing: false,
	}
}

func (r *Runner) channelReadCommand() *cli.Command {
	return &cli.Command{
		Name:      "read",
		Usage:     "Read the whole channel (self-describing onboarding view).",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "format", Usage: "json (default), yaml, or markdown"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "channel.read",
				SkipScope: true,
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := channelID(c.Name, c.Args().Slice())
					if err != nil {
						return err
					}
					q := url.Values{}
					accept := ""
					if f := c.String("format"); f != "" {
						q.Set("format", f)
						switch f {
						case "yaml":
							accept = "application/yaml"
						case "markdown":
							accept = "text/markdown"
						}
					}
					return r.channelDo(ctx, "GET", "/agent-channel/"+url.PathEscape(id), q, nil, accept)
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) channelStateCommand() *cli.Command {
	return &cli.Command{
		Name:      "state",
		Usage:     "Read the newest state event's payload.",
		ArgsUsage: "<id>",
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "channel.state",
				SkipScope: true,
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := channelID(c.Name, c.Args().Slice())
					if err != nil {
						return err
					}
					return r.channelDo(ctx, "GET", "/agent-channel/"+url.PathEscape(id)+"/state", nil, nil, "")
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) channelSpecCommand() *cli.Command {
	return &cli.Command{
		Name:      "spec",
		Usage:     "Read the newest spec event's payload (the channel charter).",
		ArgsUsage: "<id>",
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "channel.spec",
				SkipScope: true,
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := channelID(c.Name, c.Args().Slice())
					if err != nil {
						return err
					}
					return r.channelDo(ctx, "GET", "/agent-channel/"+url.PathEscape(id)+"/spec", nil, nil, "")
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) channelEventsCommand() *cli.Command {
	return &cli.Command{
		Name:      "events",
		Usage:     "List a channel's events (newest first), optionally filtered by kind.",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "kind", Usage: "filter by event kind"},
			&cli.IntFlag{Name: "limit", Usage: "max events to return (server caps at 500)"},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "channel.events",
				SkipScope: true,
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := channelID(c.Name, c.Args().Slice())
					if err != nil {
						return err
					}
					q := url.Values{}
					if k := c.String("kind"); k != "" {
						q.Set("kind", k)
					}
					if l := c.Int("limit"); l > 0 {
						q.Set("limit", fmt.Sprintf("%d", l))
					}
					return r.channelDo(ctx, "GET", "/agent-channel/"+url.PathEscape(id)+"/events", q, nil, "")
				},
			},
			r.Audit,
		),
	}
}

func (r *Runner) channelCloseCommand() *cli.Command {
	return &cli.Command{
		Name:      "close",
		Usage:     "Mark a channel closed. Events stay readable.",
		ArgsUsage: "<id>",
		Action: r.WrapVerb(
			verb.Spec{
				Name: "channel.close",
				Action: func(ctx context.Context, c *cli.Command) error {
					id, err := channelID(c.Name, c.Args().Slice())
					if err != nil {
						return err
					}
					return r.channelDo(ctx, "POST", "/agent-channel/"+url.PathEscape(id)+"/close", nil, nil, "")
				},
			},
			r.Audit,
		),
	}
}

// channelID extracts and validates the single <id> positional. Channel
// ids are 4-character uppercase tokens from the dictatable alphabet; the
// server normalizes case, but we sanity-check shape early so a typo doesn't
// look like a network error.
func channelID(verbName string, args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("channel %s: expected exactly one <id> arg, got %d", verbName, len(args))
	}
	id := strings.TrimSpace(args[0])
	if id == "" {
		return "", fmt.Errorf("channel %s: <id> is empty", verbName)
	}
	if len(id) > 16 {
		return "", fmt.Errorf("channel %s: <id> too long (got %d, channel ids are 4 chars)", verbName, len(id))
	}
	for _, ch := range id {
		switch {
		case ch >= 'A' && ch <= 'Z',
			ch >= 'a' && ch <= 'z',
			ch >= '0' && ch <= '9',
			ch == '-', ch == '_':
		default:
			return "", fmt.Errorf("channel %s: <id> contains invalid character %q", verbName, ch)
		}
	}
	return id, nil
}

// readChannelPayload mirrors the sentry wrapper's --body contract:
// empty -> nil JSON object, '-' -> stdin, '@path' -> file, otherwise
// the literal string is parsed as JSON. Returning a decoded any (not
// raw bytes) lets the caller assemble one well-formed request body
// instead of relying on string concatenation.
func readChannelPayload(spec string) (any, error) {
	var raw []byte
	switch {
	case spec == "":
		return map[string]any{}, nil
	case spec == "-":
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("channel post: read stdin: %w", err)
		}
		raw = b
	case strings.HasPrefix(spec, "@"):
		b, err := os.ReadFile(spec[1:]) // #nosec G304 -- operator-supplied path, same shape as sentry wrapper.
		if err != nil {
			return nil, fmt.Errorf("channel post: read %s: %w", spec[1:], err)
		}
		raw = b
	default:
		raw = []byte(spec)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("channel post: payload is not valid JSON: %w", err)
	}
	return decoded, nil
}

// channelDo issues one HTTP request against the configured Agent Channel
// deployment, attaching the SSM-resolved bearer token. Response is
// streamed verbatim when accept names yaml/markdown (the server already
// formats it); JSON responses are pretty-printed.
func (r *Runner) channelDo(ctx context.Context, method, path string, q url.Values, body []byte, accept string) error {
	base := strings.TrimRight(r.Cfg.Channel.BaseURL, "/")
	if base == "" {
		return fmt.Errorf("channel: channel.base_url not set in config")
	}
	tokenPath := r.Cfg.Channel.SSMTokenPath
	if tokenPath == "" {
		return fmt.Errorf("channel: channel.ssm_token_path not set in config")
	}
	target := base + path
	if len(q) > 0 {
		target += "?" + q.Encode()
	}
	var rdr io.Reader
	if len(body) > 0 {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, rdr)
	if err != nil {
		return fmt.Errorf("channel: build request: %w", err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	} else {
		req.Header.Set("Accept", "application/json")
	}
	token, err := r.channelToken(ctx, tokenPath)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: channelHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("channel: %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("channel: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("channel: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return renderChannelResponse(respBody, resp.Header.Get("Content-Type"))
}

// renderChannelResponse pretty-prints JSON responses and streams everything
// else verbatim. Split out from channelDo to keep that function under the
// gocyclo cap and to keep the rendering branches independently testable.
func renderChannelResponse(body []byte, contentType string) error {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	if !strings.Contains(contentType, "json") {
		if _, err := os.Stdout.Write(body); err != nil {
			return err
		}
		if !bytes.HasSuffix(body, []byte("\n")) {
			if _, err := fmt.Fprintln(os.Stdout); err != nil {
				return err
			}
		}
		return nil
	}
	var pretty json.RawMessage
	if err := json.Unmarshal(body, &pretty); err != nil {
		_, werr := os.Stdout.Write(body)
		return werr
	}
	out, err := json.MarshalIndent(pretty, "", "  ")
	if err != nil {
		return fmt.Errorf("channel: re-encode response: %w", err)
	}
	_, err = fmt.Fprintln(os.Stdout, string(out))
	return err
}

// channelToken resolves the bearer token from AWS SSM. Uses r.Runner.Capture
// (same path as the sentry/trello wrappers) so the aws CLI binary is found
// via $PATH and credentials come from the operator's resolved profile.
func (r *Runner) channelToken(ctx context.Context, ssmPath string) (string, error) {
	out, err := r.Runner.Capture(ctx, "aws",
		"ssm", "get-parameter",
		"--name", ssmPath,
		"--with-decryption",
		"--query", "Parameter.Value",
		"--output", "text",
	)
	if err != nil {
		return "", fmt.Errorf("channel: fetch %s: %w", ssmPath, err)
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return "", fmt.Errorf("channel: %s resolved to empty value", ssmPath)
	}
	return v, nil
}
