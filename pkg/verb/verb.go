// Package verb is the middleware that wraps every coily command action in
// the standard pipeline of:
//
//  1. Argument validation (no shell metacharacters).
//  2. Action execution.
//  3. Audit-log record.
//
// Using verb.Wrap is the way coily guarantees that every user-visible verb
// goes through the security boundary. Anything that constructs a
// *cli.Command.Action by hand bypasses audit logging and argv validation.
// Don't do that.
package verb

import (
	"context"
	"fmt"
	"os"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/exitcode"
	"github.com/coilysiren/coily/pkg/policy"
	"github.com/coilysiren/coily/pkg/scope"
	"github.com/urfave/cli/v3"
)

// CommitScopeFlag is the canonical name of the global --commit-scope flag.
// Exported so cmd/coily can declare the flag and verb.Wrap can read it
// without disagreeing on spelling.
const CommitScopeFlag = "commit-scope"

// Spec describes a verb before it is wrapped into a cli.ActionFunc.
type Spec struct {
	// Name is the dotted verb path used for audit logging, e.g.
	// "aws.route53.change-resource-record-sets" or "lockdown".
	Name string

	// ArgsFunc extracts the user-supplied string arguments from the
	// *cli.Command for validation. Returns named flags and positional args.
	// Both are fed to policy.ValidateArgs / ValidateArgSlice. If ArgsFunc is
	// nil, Wrap treats the verb as having no user-supplied string input.
	ArgsFunc func(*cli.Command) (args map[string]string, positional []string)

	// Action is the verb's real work. Called only after argv validation passes.
	Action cli.ActionFunc

	// SkipPolicy disables the shell-metacharacter check for this verb. Set
	// true only for pass-throughs whose argv goes straight through execve to
	// a tool that does not feed it back through a shell (gh, aws, tailscale,
	// package managers). The audit log and the lockdown deny list still
	// cover the boundary; the metacharacter check is paranoia for the
	// remote-shell path (ssh remote-cmd, kubectl/docker exec into bash -c)
	// and gets in the way of legitimate argv content like markdown bodies
	// (backticks, '>', '$') that callers want to forward verbatim.
	SkipPolicy bool

	// SkipScope disables --commit-scope resolution for this verb. Set true
	// for read-only or self-referential verbs that would refuse to run
	// outside a git repo otherwise (version, whoami, audit, git trailer/
	// audit-show, lockdown, setup, install-completion). The audit row is
	// still written; CommitScope is just left empty so the row never appears
	// in any commit's trailer query.
	SkipScope bool

	// OnComplete, if set, runs inside writer.Wrap after Action returns and
	// before the audit record is appended. Receives a pointer to the record
	// being finalized so the verb can attach side-channel data (e.g. the
	// rows collected by the egress proxy in pkg/egress). Decision /
	// ExitCode / DurationMS / Error are already set when OnComplete runs;
	// mutating them is not the contract.
	OnComplete func(*audit.Record)

	// CommitScopeOverride, when non-empty, replaces flag/env resolution and
	// uses this absolute path as the audit row's commit-scope. Set by `coily
	// exec` discovered-from-child verbs so audit rows bind to the matched
	// child repo, not cwd's git toplevel (which often is not a repo when the
	// operator runs `coily exec` one directory above the target). Ignored
	// when SkipScope is true.
	CommitScopeOverride string

	// CommitScopeArgvHint, when set, runs as a fallback resolver before
	// scope.Resolve and only when neither --commit-scope (still at "auto")
	// nor $COILY_COMMIT_SCOPE was set explicitly. Receives the verb's argv;
	// returning a non-empty path becomes the resolved commit-scope. Used by
	// `coily ops gh` to default the scope to ~/projects/coilysiren/<name>
	// when the user passed --repo coilysiren/<name>. Loses to
	// CommitScopeOverride and to any explicit flag/env value. Ignored when
	// SkipScope is true.
	CommitScopeArgvHint func(argv []string) string
}

// Wrap returns a cli.ActionFunc that runs the full coily verb pipeline.
//
// writer may be nil in dev contexts; a nil writer skips audit logging.
func Wrap(spec Spec, writer *audit.Writer) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// os.Args is what the user typed. Better for audit than trying to
		// reconstruct from cli.Command state (which requires a fully-
		// initialized cmd and is awkward to assemble).
		argv := append([]string{}, os.Args...)
		if !spec.SkipPolicy {
			args, positional := extractArgs(spec, cmd)
			if err := policy.ValidateArgs(args); err != nil {
				coded := exitcode.New(exitcode.PolicyDenied, "policy_denied", err,
					"argv contains a shell metacharacter that coily refuses to forward")
				logReject(writer, spec.Name, argv, coded)
				return coded
			}
			if err := policy.ValidateArgSlice("positional", positional); err != nil {
				coded := exitcode.New(exitcode.PolicyDenied, "policy_denied", err,
					"a positional argument failed shell-metacharacter validation")
				logReject(writer, spec.Name, argv, coded)
				return coded
			}
		}

		base, scopeErr := buildBaseRecord(spec, argv, cmd)
		if scopeErr != nil {
			coded := exitcode.New(exitcode.Generic, "scope_unresolved", scopeErr,
				"set --commit-scope=<repo-path> or COILY_COMMIT_SCOPE=<repo-path>; "+
					"there is no opt-out, every audit row must bind to a real repo")
			logReject(writer, spec.Name, argv, coded)
			return coded
		}

		if writer == nil {
			return spec.Action(ctx, cmd)
		}
		return writer.WrapHook(ctx, base, func() error {
			return spec.Action(ctx, cmd)
		}, spec.OnComplete)
	}
}

// buildBaseRecord composes the per-invocation Record that writer.Wrap will
// fill in with Decision/ExitCode/DurationMS. Resolves --commit-scope here
// so a misconfigured shell fails loud before fn runs. Honors
// spec.CommitScopeOverride when set so verbs that pre-compute their
// commit-scope (notably `coily exec` from a direct-child match) can bind
// the audit row to a path that is not cwd's git toplevel. argv is the full
// os.Args captured by Wrap and is fed to spec.CommitScopeArgvHint when
// neither --commit-scope nor $COILY_COMMIT_SCOPE was set explicitly.
func buildBaseRecord(spec Spec, argv []string, cmd *cli.Command) (audit.Record, error) {
	cwd := scope.CWD()
	repoRoot, _ := scope.Resolve("auto", "", cwd) // forensic-only, ignore error
	if spec.SkipScope {
		return audit.Record{
			Verb:     spec.Name,
			Argv:     argv,
			RepoRoot: repoRoot,
		}, nil
	}
	if spec.CommitScopeOverride != "" {
		return audit.Record{
			Verb:        spec.Name,
			Argv:        argv,
			RepoRoot:    repoRoot,
			CommitScope: spec.CommitScopeOverride,
		}, nil
	}
	root := cmd
	if r := cmd.Root(); r != nil {
		root = r
	}
	flagVal := root.String(CommitScopeFlag)
	envVal := os.Getenv("COILY_COMMIT_SCOPE")
	if (flagVal == "" || flagVal == "auto") && envVal == "" && spec.CommitScopeArgvHint != nil {
		if hint := spec.CommitScopeArgvHint(argv); hint != "" {
			flagVal = hint
		}
	}
	commitScope, err := scope.Resolve(flagVal, envVal, cwd)
	if err != nil {
		return audit.Record{}, err
	}
	return audit.Record{
		Verb:        spec.Name,
		Argv:        argv,
		RepoRoot:    repoRoot,
		CommitScope: commitScope,
	}, nil
}

func logReject(writer *audit.Writer, verbName string, argv []string, err error) {
	if writer == nil {
		return
	}
	rec := audit.Record{
		Decision: audit.DecisionReject,
		Verb:     verbName,
		Argv:     argv,
		ExitCode: 1,
		Error:    err.Error(),
	}
	if aerr := writer.Append(rec); aerr != nil {
		fmt.Fprintf(os.Stderr, "audit: %v\n", aerr)
	}
}

func extractArgs(spec Spec, cmd *cli.Command) (args map[string]string, positional []string) {
	if spec.ArgsFunc == nil {
		return nil, nil
	}
	return spec.ArgsFunc(cmd)
}
