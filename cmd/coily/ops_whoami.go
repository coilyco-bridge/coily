package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/coilysiren/cli-guard/ghidcache"
	"github.com/coilysiren/cli-guard/shell"
	"github.com/coilysiren/cli-guard/stscache"
	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

func (r *Runner) whoamiCommand() *cli.Command {
	return &cli.Command{
		Name:  "whoami",
		Usage: "Print the authenticated identity coily sees across aws, kubectl, gh, plus the agent self-name.",
		Description: `whoami asks each underlying tool "who am I?" and unifies the results into a
single yaml block. Useful for confirming which AWS account coily is pointed
at before a write op, seeing which kubectl context is current, and checking
which GitHub identity gh is authenticated as.

The agent block prints this agent's self-name in the form
claude-<os>-<hostname>-<tag>-<pronouns>, where <tag> is a stable 4-char
dictatable id (two letters then two digits) derived from the Claude Code
session id and <pronouns> is the agent's pronoun slug (she-her for Claude).
Codex and OpenClaw swap the claude- prefix and their pronouns - codex
he-him, openclaw they-them.

Non-fatal. If any tool is not configured or returns an error, its block
reports the error but the other tools still run.`,
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "whoami",
				SkipScope: true,
				Action:    r.whoamiAction,
			},
			r.Audit,
		),
	}
}

func (r *Runner) whoamiAction(ctx context.Context, _ *cli.Command) error {
	out := map[string]any{
		"aws":     awsWhoami(ctx, r.Runner),
		"kubectl": kubectlWhoami(ctx, r.Runner),
		"gh":      ghWhoami(ctx, r.Runner),
		"agent":   agentWhoami(),
	}
	b, err := yaml.Marshal(out)
	if err != nil {
		return err
	}
	fmt.Print(string(b))
	return nil
}

func awsWhoami(ctx context.Context, r *shell.Runner) any {
	raw, err := stscache.CallerIdentity(func() ([]byte, error) {
		return r.Capture(ctx, "aws", "sts", "get-caller-identity", "--output", "json")
	})
	if err != nil {
		return errBlock("aws sts get-caller-identity failed", err)
	}
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		return errBlock("aws sts output was not JSON", err)
	}
	return v
}

func kubectlWhoami(ctx context.Context, r *shell.Runner) any {
	// Prefer `kubectl auth whoami` (kubectl 1.28+). Falls back to parsing
	// the minified config, which is what older kubectl installs support.
	if raw, err := r.Capture(ctx, "kubectl", "auth", "whoami", "-o", "json"); err == nil {
		var v map[string]any
		if err := json.Unmarshal(raw, &v); err == nil {
			return v
		}
	}

	ctxRaw, cerr := r.Capture(ctx, "kubectl", "config", "current-context")
	if cerr != nil {
		return errBlock("kubectl config current-context failed", cerr)
	}
	out := map[string]any{"current_context": strings.TrimSpace(string(ctxRaw))}

	return enrichKubectlContext(ctx, r, out)
}

func enrichKubectlContext(ctx context.Context, r *shell.Runner, out map[string]any) map[string]any {
	cfgRaw, verr := r.Capture(ctx, "kubectl", "config", "view", "--minify", "-o", "json")
	if verr != nil {
		return out
	}
	var cfg map[string]any
	if err := json.Unmarshal(cfgRaw, &cfg); err != nil {
		return out
	}
	contexts, _ := cfg["contexts"].([]any)
	if len(contexts) == 0 {
		return out
	}
	c, _ := contexts[0].(map[string]any)
	cc, _ := c["context"].(map[string]any)
	if cc == nil {
		return out
	}
	for _, k := range []string{"cluster", "user", "namespace"} {
		if v, ok := cc[k]; ok {
			out[k] = v
		}
	}
	return out
}

func ghWhoami(ctx context.Context, r *shell.Runner) any {
	raw, err := ghidcache.APIUser(func() ([]byte, error) {
		return r.Capture(ctx, "gh", "api", "user")
	})
	if err != nil {
		return errBlock("gh api user failed", err)
	}
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		return errBlock("gh api user output was not JSON", err)
	}
	// Strip fields that aren't identity-relevant. gh api user returns ~40
	// fields, most of them profile/statistics noise.
	keep := map[string]bool{
		"login":    true,
		"id":       true,
		"name":     true,
		"email":    true,
		"html_url": true,
		"type":     true,
	}
	out := map[string]any{}
	for k, val := range v {
		if keep[k] {
			out[k] = val
		}
	}
	return out
}

// agentIdentity is this agent's self-name and the parts it is built from.
type agentIdentity struct {
	name, agent, osSlug, hostname, tag, pronouns string
}

// agentPronounSlug maps an agent to its pronoun slug, the trailing segment of
// the self-name. Slug form (dash, not slash) so it stays a valid name token:
// claude she-her, codex he-him. Everything else - openclaw and any
// unrecognized harness - defaults to they-them.
func agentPronounSlug(agent string) string {
	switch agent {
	case "claude":
		return "she-her"
	case "codex":
		return "he-him"
	default: // openclaw and any unrecognized harness
		return "they-them"
	}
}

// resolveAgentIdentity builds the agent self-name from a session id:
// claude-<os>-<hostname>-<tag>-<pronouns>. The tag is dropped when sessionID
// is empty; the pronoun slug is always present. Pure local lookup, no
// subprocess, so it never errors. This is the canonical name computation -
// the agent-name verb and the whoami agent block both route through here.
func resolveAgentIdentity(sessionID string) agentIdentity {
	osSlug := agentOSSlug(runtime.GOOS)
	host := agentShortHostname()
	tag := agentSessionTag(sessionID)
	agent := "claude"
	pronouns := agentPronounSlug(agent)
	parts := []string{agent, osSlug, host}
	if tag != "" {
		parts = append(parts, tag)
	}
	parts = append(parts, pronouns)
	return agentIdentity{
		name:     strings.Join(parts, "-"),
		agent:    agent,
		osSlug:   osSlug,
		hostname: host,
		tag:      tag,
		pronouns: pronouns,
	}
}

// agentWhoami builds the agent self-name block for whoami output. The
// session id comes from $CLAUDE_CODE_SESSION_ID, set in Bash-tool context.
func agentWhoami() any {
	id := resolveAgentIdentity(os.Getenv(sessionEnvVar))
	return map[string]any{
		"name":        id.name,
		"agent":       id.agent,
		"os":          id.osSlug,
		"hostname":    id.hostname,
		"session_tag": id.tag,
		"pronouns":    id.pronouns,
	}
}

// agentOSSlug maps runtime.GOOS to the friendly slug used in the name.
func agentOSSlug(goos string) string {
	switch goos {
	case "darwin":
		return "macos"
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	default:
		return goos
	}
}

// agentShortHostname returns the lowercased hostname with any domain stripped.
func agentShortHostname() string {
	h, err := os.Hostname()
	if err != nil || strings.TrimSpace(h) == "" {
		return "unknown-host"
	}
	h = strings.ToLower(strings.TrimSpace(h))
	if i := strings.IndexByte(h, '.'); i >= 0 {
		h = h[:i]
	}
	return h
}

// Dictatable alphabet, lowercased to fit the all-lowercase agent name. Mirrors
// the otel-a2a-relay channels generator (otel_a2a_relay_channels/ids.py) and
// agentic-os docs/dictatable-id-alphabet.md, which drop the visually and
// phonetically ambiguous characters from base32/base58.
const (
	tagLetters = "abcdefghjkmpqrstuvwxyz"
	tagDigits  = "456789"
)

// agentSessionTag derives the stable per-session suffix of the agent name: a
// dictatable id shaped as two letters then two digits (e.g. "ab45"), the same
// shape o2r and backend channel ids use. Deterministic in the session id, so a
// session always self-names the same way; the SHA-256 digest spreads adjacent
// session ids apart instead of slicing raw UUID hex. Empty id yields an empty
// tag, and the suffix is then dropped.
func agentSessionTag(sessionID string) string {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return ""
	}
	h := sha256.Sum256([]byte(sid))
	return string([]byte{
		tagLetters[int(h[0])%len(tagLetters)],
		tagLetters[int(h[1])%len(tagLetters)],
		tagDigits[int(h[2])%len(tagDigits)],
		tagDigits[int(h[3])%len(tagDigits)],
	})
}

func errBlock(msg string, err error) map[string]string {
	return map[string]string{"error": fmt.Sprintf("%s: %v", msg, err)}
}
