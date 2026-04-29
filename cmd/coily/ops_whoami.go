package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coilysiren/coily/pkg/shell"
	"github.com/coilysiren/coily/pkg/verb"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

func (r *Runner) whoamiCommand() *cli.Command {
	return &cli.Command{
		Name:  "whoami",
		Usage: "Print the authenticated identity coily sees across aws, kubectl, and gh.",
		Description: `whoami asks each underlying tool "who am I?" and unifies the results into a
single yaml block. Useful for confirming which AWS account coily is pointed
at before a write op, seeing which kubectl context is current, and checking
which GitHub identity gh is authenticated as.

Non-fatal. If any tool is not configured or returns an error, its block
reports the error but the other tools still run.`,
		Action: verb.Wrap(
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
	}
	b, err := yaml.Marshal(out)
	if err != nil {
		return err
	}
	fmt.Print(string(b))
	return nil
}

func awsWhoami(ctx context.Context, r *shell.Runner) any {
	raw, err := r.Capture(ctx, "aws", "sts", "get-caller-identity", "--output", "json")
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
	raw, err := r.Capture(ctx, "gh", "api", "user")
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

func errBlock(msg string, err error) map[string]string {
	return map[string]string{"error": fmt.Sprintf("%s: %v", msg, err)}
}
