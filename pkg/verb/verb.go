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
	"github.com/coilysiren/coily/pkg/policy"
	"github.com/urfave/cli/v3"
)

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
		args, positional := extractArgs(spec, cmd)
		if err := policy.ValidateArgs(args); err != nil {
			logReject(writer, spec.Name, argv, err)
			return err
		}
		if err := policy.ValidateArgSlice("positional", positional); err != nil {
			logReject(writer, spec.Name, argv, err)
			return err
		}

		if writer == nil {
			return spec.Action(ctx, cmd)
		}
		return writer.Wrap(ctx, spec.Name, argv, func() error {
			return spec.Action(ctx, cmd)
		})
	}
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
