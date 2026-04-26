package skillgen_test

import (
	"strings"
	"testing"

	"github.com/coilysiren/coily/pkg/skillgen"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

func sampleTree() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "aws",
			Usage: "Pass-through to aws.",
			Commands: []*cli.Command{
				{
					Name:  "ssm",
					Usage: "ssm",
					Commands: []*cli.Command{
						{
							Name:  "get-parameter",
							Usage: "Read one parameter.",
							Flags: []cli.Flag{
								&cli.StringFlag{Name: "name"},
								&cli.BoolFlag{Name: "with-decryption"},
							},
						},
					},
				},
			},
		},
	}
}

func TestRenderPassthroughs_ContainsLeaf(t *testing.T) {
	body := skillgen.RenderPassthroughs(sampleTree())
	if !strings.Contains(body, "## `coily aws ssm get-parameter`") {
		t.Error("markdown body missing leaf header")
	}
	if !strings.Contains(body, "Flags: --name, --with-decryption") {
		t.Error("markdown body missing flags line")
	}
}

func TestRenderPassthroughsYAML_StructuredShape(t *testing.T) {
	body, err := skillgen.RenderPassthroughsYAML(sampleTree())
	if err != nil {
		t.Fatalf("RenderPassthroughsYAML: %v", err)
	}
	var parsed struct {
		Commands []skillgen.PassthroughEntry `yaml:"commands"`
	}
	if err := yaml.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(parsed.Commands) != 1 {
		t.Fatalf("got %d commands, want 1", len(parsed.Commands))
	}
	got := parsed.Commands[0]
	wantPath := []string{"coily", "aws", "ssm", "get-parameter"}
	if strings.Join(got.Path, ".") != strings.Join(wantPath, ".") {
		t.Errorf("path = %v, want %v", got.Path, wantPath)
	}
	if got.Summary != "Read one parameter." {
		t.Errorf("summary = %q", got.Summary)
	}
	if len(got.Flags) != 2 {
		t.Errorf("flags = %v, want 2", got.Flags)
	}
}
