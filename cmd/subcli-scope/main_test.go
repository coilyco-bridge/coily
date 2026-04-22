package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// updateGolden controls whether the tests rewrite the committed goldens
// instead of asserting against them. Set via `go test -update`.
var updateGolden = flag.Bool("update", false, "rewrite golden files instead of comparing")

// TestScanAgainstFixtures runs the parser against captured help-text fixtures
// for each target CLI and compares the resulting manifest to a committed
// golden. This catches two classes of regression:
//
//   - Our parser breaking on a help format we used to handle.
//   - A target CLI shipping a help-format change that our parser misses.
//     (Caught when fixtures are refreshed via `make update-fixtures` and the
//     golden diff reveals disappearances or new shape.)
//
// To refresh fixtures from live CLIs, run `make update-fixtures`.
// To refresh goldens after an intentional parser change, run
// `go test ./cmd/subcli-scope -update`.
func TestScanAgainstFixtures(t *testing.T) {
	cases := []struct {
		binary string
	}{
		{"aws"},
		{"gh"},
		{"kubectl"},
	}

	for _, tc := range cases {
		t.Run(tc.binary, func(t *testing.T) {
			scope, err := loadScopeForTest(tc.binary)
			if err != nil {
				t.Fatalf("load scope: %v", err)
			}

			fixtureDir := filepath.Join("testdata", "fixtures")
			if _, err := os.Stat(filepath.Join(fixtureDir, tc.binary, "help.txt")); err != nil {
				t.Skipf("no fixture captured for %s; run `make update-fixtures` to capture",
					tc.binary)
			}

			got := Scan(tc.binary, scope, fixtureFetcher(fixtureDir))

			goldenPath := filepath.Join("testdata", tc.binary+".golden.yaml")
			gotYAML, err := yaml.Marshal(struct {
				Commands []Command `yaml:"commands"`
			}{Commands: got})
			if err != nil {
				t.Fatalf("marshal got: %v", err)
			}

			if *updateGolden {
				if err := os.WriteFile(goldenPath, gotYAML, 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				t.Logf("wrote %s", goldenPath)
				return
			}

			wantBytes, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden %s: %v (run with -update to create)", goldenPath, err)
			}
			var want struct {
				Commands []Command `yaml:"commands"`
			}
			if err := yaml.Unmarshal(wantBytes, &want); err != nil {
				t.Fatalf("parse golden: %v", err)
			}
			wantYAML, _ := yaml.Marshal(want)

			if string(gotYAML) != string(wantYAML) {
				t.Errorf("manifest differs from golden for %s.\n"+
					"run `go test ./cmd/subcli-scope -update` if the change is intentional.\n"+
					"first divergent region follows:\n%s",
					tc.binary, firstDiff(string(wantYAML), string(gotYAML)))
			}
		})
	}
}

// TestExtractSubcommandsShapes is a unit test on the parser against tiny
// hand-written help snippets. Keeps the shape of the parser honest even if
// the fixtures drift.
func TestExtractSubcommandsShapes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "gh colon style",
			in: `USAGE
  gh <command>

CORE COMMANDS
  auth:        Authenticate gh and git with GitHub
  browse:      Open repositories
  repo:        Manage repositories

FLAGS
  --help
`,
			want: []string{"auth", "browse", "repo"},
		},
		{
			name: "kubectl space style",
			in: `Usage:
  kubectl [flags] [options]

Available Commands:
  get          Display one or many resources
  describe     Show details of a specific resource

Use "kubectl <command> --help" for more information about a given command.
`,
			want: []string{"get", "describe"},
		},
		{
			name: "aws bullet style",
			in: `AVAILABLE SERVICES
       o route53

       o s3

       o ssm
`,
			want: []string{"route53", "s3", "ssm"},
		},
		{
			name: "noise filtered (help, version, completion)",
			in: `COMMANDS
  deploy       Deploy something
  help         Show help
  version      Show version
  completion   Shell completion
`,
			want: []string{"deploy"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractSubcommands(tc.in)
			if !equalSlices(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestStripOverstrike checks groff-style backspace-overstrike removal used
// when ingesting aws help (which renders through man-style formatting).
func TestStripOverstrike(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"N\bNAA", "NAA"},
		{"N\bN", "N"},
		{"normal", "normal"},
		{"", ""},
		{"A\bA\bA", "A"},
	}
	for _, tc := range cases {
		got := stripOverstrike(tc.in)
		if got != tc.want {
			t.Errorf("stripOverstrike(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// loadScopeForTest reads configs/scopes/<bin>.yaml relative to the repo root.
// Tests run with CWD set to the package dir (cmd/subcli-scope), so we climb
// up two levels.
func loadScopeForTest(bin string) (*Scope, error) {
	path := filepath.Join("..", "..", "configs", "scopes", bin+".yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Scope
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// firstDiff returns a small window around the first differing line between a
// and b, to give the test failure a useful context without dumping the entire
// manifest.
func firstDiff(a, b string) string {
	const window = 400
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	for i := 0; i < minLen; i++ {
		if a[i] != b[i] {
			start := i - 100
			if start < 0 {
				start = 0
			}
			endA := i + window
			if endA > len(a) {
				endA = len(a)
			}
			endB := i + window
			if endB > len(b) {
				endB = len(b)
			}
			return "WANT:\n" + a[start:endA] + "\n\nGOT:\n" + b[start:endB]
		}
	}
	if len(a) != len(b) {
		return "lengths differ (want=" + itoa(len(a)) + " got=" + itoa(len(b)) + ")"
	}
	return "(no textual diff but marshaled strings compared unequal)"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
