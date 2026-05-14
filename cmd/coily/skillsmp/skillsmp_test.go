package skillsmp

import (
	"testing"

	"github.com/coilysiren/cli-guard/audit"
	"github.com/coilysiren/cli-guard/shell"
	"github.com/urfave/cli/v3"
)

// TestCommand_AcceptsQueryAndOutputFlags pins coilysiren/coily#172: every
// leaf carries the aws-CLI-style --query and --output projection +
// format flags. A new leaf added without the pair would fail this gate.
func TestCommand_AcceptsQueryAndOutputFlags(t *testing.T) {
	cmd := Command(&shell.Runner{}, &audit.Writer{})
	for _, sub := range cmd.Commands {
		t.Run(sub.Name, func(t *testing.T) {
			has := map[string]bool{}
			for _, f := range sub.Flags {
				for _, n := range f.Names() {
					has[n] = true
				}
			}
			for _, want := range []string{"query", "output"} {
				if !has[want] {
					t.Errorf("leaf %q missing --%s flag (got flags %v)", sub.Name, want, flagNames(sub))
				}
			}
		})
	}
}

func flagNames(c *cli.Command) []string {
	var out []string
	for _, f := range c.Flags {
		out = append(out, f.Names()...)
	}
	return out
}
