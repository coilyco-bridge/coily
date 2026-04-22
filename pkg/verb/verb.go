// Package verb is the middleware that wraps every coily command action in
// the standard pipeline of:
//
//  1. Policy enforcement (argv validation + confirmation-token check).
//  2. Action execution.
//  3. Audit-log record.
//
// Using verb.Wrap is the way coily guarantees that every user-visible verb
// goes through the security boundary. Anything that constructs a
// *cli.Command.Action by hand bypasses audit logging and policy checks.
// Don't do that.
package verb

import (
	"context"
	"os"

	"github.com/coilysiren/coily/pkg/audit"
	"github.com/coilysiren/coily/pkg/policy"
	"github.com/urfave/cli/v3"
)

// Spec describes a verb before it is wrapped into a cli.ActionFunc.
type Spec struct {
	// Name is the dotted verb path for audit logging and token scope, e.g.
	// "aws.route53.change-resource-record-sets" or "lockdown".
	Name string

	// Kind is ReadOnly or Mutating. Mutating verbs require a confirmation token.
	Kind policy.Kind

	// ArgsFunc extracts the user-supplied string arguments from the
	// *cli.Command. Returns three parts:
	//
	//   - args: named flags, e.g. {"--hosted-zone-id": "Z123"}.
	//   - positional: positional args in order.
	//   - token: the confirmation token if present (COILY_TOKEN env or a
	//     --token flag), or "" if none.
	//
	// All returned strings are fed to policy.Enforce. If ArgsFunc is nil,
	// Wrap treats the verb as having no user-supplied string input.
	ArgsFunc func(*cli.Command) (args map[string]string, positional []string, token string)

	// Action is the verb's real work. Called only after policy.Enforce passes.
	Action cli.ActionFunc
}

// Wrap returns a cli.ActionFunc that runs the full coily verb pipeline.
//
// Either writer or verifier may be nil in dev contexts. A nil writer skips
// audit logging with a warning. A nil verifier is only safe for ReadOnly
// verbs; Mutating verbs with a nil verifier always fail policy.Enforce.
func Wrap(spec Spec, verifier policy.TokenVerifier, writer *audit.Writer) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		args, positional, token := extractArgs(spec, cmd)
		inv := policy.Invocation{
			Verb:       spec.Name,
			Kind:       spec.Kind,
			Args:       args,
			Positional: positional,
			Token:      token,
		}
		if err := policy.Enforce(inv, verifier); err != nil {
			return err
		}

		if writer == nil {
			return spec.Action(ctx, cmd)
		}
		// os.Args is what the user typed. Better for audit than trying to
		// reconstruct from cli.Command state (which requires a fully-
		// initialized cmd and is awkward to assemble).
		argv := append([]string{}, os.Args...)
		return writer.Wrap(ctx, spec.Name, argv, func() error {
			return spec.Action(ctx, cmd)
		})
	}
}

func extractArgs(spec Spec, cmd *cli.Command) (args map[string]string, positional []string, token string) {
	if spec.ArgsFunc == nil {
		return nil, nil, ""
	}
	return spec.ArgsFunc(cmd)
}
