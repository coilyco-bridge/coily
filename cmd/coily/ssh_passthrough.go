package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/coilysiren/cli-guard/audit"
	"github.com/coilysiren/cli-guard/repocfg"
	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// sshPassthroughAction handles `coily ssh <host-alias> -- <coily args>`.
//
// The parent ssh command falls through here when none of the named subcommands
// (systemctl, copy, deploy, ...) matched the first positional arg, so the
// per-verb wrappers and the passthrough coexist during the migration window
// laid out in coilysiren/coily#187. Long-term plan is to deprecate the
// per-verb wrappers; for now both paths share the parent.
//
// Resolution flow:
//
//  1. Read the first positional as the host alias.
//  2. urfave's flag parser already swallowed the literal `--` separator;
//     the remaining positionals are the remote coily argv.
//  3. Discover .coily/coily.yaml via repocfg.DiscoverAll, find the first
//     config whose ssh.targets carries this alias.
//  4. Pre-allocate the local audit-row id with audit.NewUUIDv7 so the same
//     id can be both the local Spec.IDOverride AND the remote
//     --audit-parent. Forensic walks (forward and backward) line up.
//  5. Build the remote command: `coily --commit-scope=<wd>
//     --audit-parent=<id> <rest...>`, POSIX-quote each token, ssh -T -stream
//     stdout/stderr back to the local terminal.
//
// SkipPolicy is set because the remote argv may carry legitimate shell
// metacharacters (markdown bodies in `gh issue create --body`, etc.). The
// metacharacters never reach a local shell - they reach the remote sshd
// through a single composed string that we control via POSIX quoting.
func (r *Runner) sshPassthroughAction(ctx context.Context, c *cli.Command) error {
	argv := c.Args().Slice()
	if len(argv) == 0 {
		return cli.ShowAppHelp(c)
	}
	alias := argv[0]
	rest := normalizePassthroughRest(argv[1:])
	if len(rest) == 0 {
		return fmt.Errorf("ssh passthrough: usage: coily ssh %s -- coily <args>", alias)
	}

	target, err := r.resolveSSHTarget(alias)
	if err != nil {
		return err
	}

	localID, err := audit.NewUUIDv7()
	if err != nil {
		return fmt.Errorf("ssh passthrough: pre-allocate audit id: %w", err)
	}

	remoteArgv := append([]string{
		"coily",
		"--cwd=" + target.WorkingDir,
		"--commit-scope=" + target.WorkingDir,
		"--audit-parent=" + localID,
	}, rest...)
	remoteCmd := joinPOSIX(remoteArgv)

	spec := verb.Spec{
		Name:       "ssh.passthrough",
		IDOverride: localID,
		SkipPolicy: true,
		ArgsFunc: func(_ *cli.Command) (map[string]string, []string) {
			return map[string]string{
					"--alias": alias,
					"--host":  target.Host,
					"--user":  target.User,
				},
				rest
		},
		Action: func(ctx context.Context, _ *cli.Command) error {
			return r.SSH.Stream(ctx, target.Host, target.User, remoteCmd, os.Stdout, os.Stderr)
		},
	}
	return r.WrapVerb(spec, r.Audit)(ctx, c)
}

// resolveSSHTarget walks every reachable .coily/coily.yaml looking for an
// ssh.targets entry under alias. Repocfg.DiscoverAll is the same discovery
// pool used by `coily exec`, so the lookup surface matches operator
// expectations (ancestor walk + direct children of cwd). Returns an actionable
// "alias not found" error that names the known aliases if any.
func (r *Runner) resolveSSHTarget(alias string) (repocfg.SSHTarget, error) {
	cwd, _ := os.Getwd()
	configs, err := repocfg.DiscoverAll(cwd)
	if err != nil {
		return repocfg.SSHTarget{}, fmt.Errorf("ssh passthrough: repocfg discovery: %w", err)
	}
	for _, cfg := range configs {
		if t, ok := cfg.SSHTargets[alias]; ok {
			return t, nil
		}
	}
	var have []string
	seen := map[string]bool{}
	for _, cfg := range configs {
		for name := range cfg.SSHTargets {
			if !seen[name] {
				have = append(have, name)
				seen[name] = true
			}
		}
	}
	sort.Strings(have)
	if len(have) == 0 {
		return repocfg.SSHTarget{}, fmt.Errorf(
			"ssh passthrough: alias %q not found; no ssh.targets block in any .coily/coily.yaml reachable from cwd. "+
				"Add an ssh.targets block to the per-repo coily.yaml (see coilysiren/coily#187)",
			alias,
		)
	}
	return repocfg.SSHTarget{}, fmt.Errorf(
		"ssh passthrough: alias %q not found; known aliases: %s",
		alias, strings.Join(have, ", "),
	)
}

// joinPOSIX POSIX-quotes each argv element and joins with spaces so the
// composed string is safe to hand to a remote shell via ssh. Mirrors the
// quoting discipline that `coily ssh kubectl` uses on its argv: any token
// that contains a character outside the conservative safe set is wrapped in
// single quotes and the '\” trick handles embedded single quotes.
func joinPOSIX(argv []string) string {
	out := make([]string, len(argv))
	for i, a := range argv {
		out[i] = posixQuote(a)
	}
	return strings.Join(out, " ")
}

// normalizePassthroughRest strips the leading "--" separator (urfave keeps
// it in c.Args() depending on how the operator typed the line) and the
// leading "coily" binary token if present. The spec form is
// `coily ssh <alias> -- coily <subcommand> <args>` and the remote command
// we build re-prepends "coily" itself; without this consume step the
// composed remote argv would be `coily ... coily <subcommand>`, which
// urfave dispatches to no subcommand and falls through to help (the bug
// caught while validating coilysiren/coily#191 step 7 of #187 live).
func normalizePassthroughRest(rest []string) []string {
	if len(rest) > 0 && rest[0] == "--" {
		rest = rest[1:]
	}
	if len(rest) > 0 && rest[0] == "coily" {
		rest = rest[1:]
	}
	return rest
}

func posixQuote(s string) string {
	if s == "" {
		return "''"
	}
	safe := true
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z',
			r >= 'a' && r <= 'z',
			r >= '0' && r <= '9',
			r == '-', r == '.', r == '/', r == '_', r == '=', r == ':', r == ',':
			// keep safe = true
		default:
			safe = false
		}
		if !safe {
			break
		}
	}
	if safe {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
