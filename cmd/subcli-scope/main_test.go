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

// writeGoldenAndManifest is the -update writer extracted out of the test
// body. It refreshes both the golden YAML (used by the assertion path) and
// the live configs/commands/<bin>.yaml manifest (consumed by the codegen),
// preserving the binary/bin_version/scanned_at metadata from the prior
// manifest so a parser-only refresh doesn't blow away the live-capture
// stamps.
func writeGoldenAndManifest(t *testing.T, binary, goldenPath string, goldenYAML []byte, commands []Command) {
	t.Helper()
	if err := os.WriteFile(goldenPath, goldenYAML, 0o644); err != nil {
		t.Fatalf("write golden: %v", err)
	}
	t.Logf("wrote %s", goldenPath)
	m := Manifest{
		Binary:    binary,
		ScannedAt: "fixture-regen",
		Commands:  commands,
	}
	manifestPath := filepath.Join("..", "..", "configs", "commands", binary+".yaml")
	prevBinVer, prevScanned := readPriorManifestStamps(manifestPath)
	if prevBinVer != "" {
		m.BinVersion = prevBinVer
	}
	if prevScanned != "" {
		m.ScannedAt = prevScanned
	}
	mb, err := yaml.Marshal(&m)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, mb, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	t.Logf("wrote %s", manifestPath)
}

// readPriorManifestStamps returns the (bin_version, scanned_at) fields from
// an existing manifest at path. Empty strings on missing-file or any parse
// error: the caller treats those as "no prior stamps" and uses defaults.
func readPriorManifestStamps(path string) (binVer, scannedAt string) {
	existing, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	var prev Manifest
	if err := yaml.Unmarshal(existing, &prev); err != nil {
		return "", ""
	}
	return prev.BinVersion, prev.ScannedAt
}

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
				writeGoldenAndManifest(t, tc.binary, goldenPath, gotYAML, got)
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

// TestClassifiedSnapshotAgainstFixtures regenerates the per-tool
// classification snapshot from the captured help fixtures and compares it
// to the committed golden. Diffing this file is the cheap way to surface a
// new verb that got the wrong READONLY/MUTATING label - the classification
// would otherwise only become visible inside generated.go, where the noise
// of regenerated commands hides it.
//
// The snapshot covers EVERY discovered verb, including ones the scope
// allow/deny filter rejects. That is the point: a filtered-out verb today
// might be in scope tomorrow, and a wrong label today rides along silently.
func TestClassifiedSnapshotAgainstFixtures(t *testing.T) {
	cases := []struct{ binary string }{{"aws"}, {"gh"}, {"kubectl"}}
	for _, tc := range cases {
		t.Run(tc.binary, func(t *testing.T) {
			fixtureDir := filepath.Join("testdata", "fixtures")
			if _, err := os.Stat(filepath.Join(fixtureDir, tc.binary, "help.txt")); err != nil {
				t.Skipf("no fixture captured for %s", tc.binary)
			}
			got := renderClassified(tc.binary, ClassifyAll(tc.binary, fixtureFetcher(fixtureDir)))
			snapPath := filepath.Join("testdata", tc.binary+".classified.txt")
			if *updateGolden {
				if err := os.WriteFile(snapPath, []byte(got), 0o644); err != nil {
					t.Fatalf("write snapshot: %v", err)
				}
				t.Logf("wrote %s", snapPath)
				return
			}
			want, err := os.ReadFile(snapPath)
			if err != nil {
				t.Fatalf("read snapshot %s: %v (run with -update to create)", snapPath, err)
			}
			if got != string(want) {
				t.Errorf("classification snapshot differs for %s.\n"+
					"run `go test ./cmd/subcli-scope -update` if intentional.\n"+
					"first divergent region:\n%s",
					tc.binary, firstDiff(string(want), got))
			}
		})
	}
}

// TestExtractFlagsTypes is a unit test on the per-dialect type detection.
// One mini-help-snippet per CLI, asserting every flag's inferred Type.
// When a help format drifts (gh adding a typed placeholder shape, kubectl
// rewriting the Options block) this is the test that should fail first.
func TestExtractFlagsTypes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []Flag
	}{
		{
			name: "aws options bool string int list",
			in: "FOO()                  FOO()\n" +
				"\nNAME\n       foo -\n" +
				"\nDESCRIPTION\n       Does foo.\n" +
				"\nOPTIONS\n" +
				"       --name (string)\n" +
				"          The name.\n" +
				"\n       --debug (boolean)\n" +
				"          Turn on debug logging.\n" +
				"\n       --max-items (integer)\n" +
				"          Cap.\n" +
				"\n       --tags (list)\n" +
				"          Tags.\n",
			want: []Flag{
				{Name: "--name", Type: "string"},
				{Name: "--debug", Type: "bool"},
				{Name: "--max-items", Type: "int"},
				{Name: "--tags", Type: "stringSlice"},
			},
		},
		{
			name: "gh flags block string bool stringSlice",
			in: "USAGE\n  gh api\n\n" +
				"FLAGS\n" +
				"  -X, --method string         The HTTP method for the request\n" +
				"  -F, --field key=value       Add a typed parameter in key=value format\n" +
				"      --paginate              Make additional HTTP requests\n",
			want: []Flag{
				{Name: "--method", Type: "string"},
				{Name: "--field", Type: "stringSlice"},
				{Name: "--paginate", Type: "bool"},
			},
		},
		{
			name: "kubectl options bool stringSlice int string",
			in: "Apply a thing.\n\n" +
				"Options:\n" +
				"    --all=false:\n\tSelect all.\n" +
				"    -f, --filename=[]:\n\tThe files.\n" +
				"    --grace-period=-1:\n\tPeriod.\n" +
				"    --dry-run='none':\n\tDry run mode.\n" +
				"\nUsage:\n  kubectl apply\n",
			want: []Flag{
				{Name: "--all", Type: "bool"},
				{Name: "--filename", Type: "stringSlice"},
				{Name: "--grace-period", Type: "int"},
				{Name: "--dry-run", Type: "string"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractFlags(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d flags, want %d.\ngot=%+v\nwant=%+v", len(got), len(tc.want), got, tc.want)
			}
			for i := range got {
				if got[i].Name != tc.want[i].Name || got[i].Type != tc.want[i].Type {
					t.Errorf("flag[%d] = %+v, want %+v", i, got[i], tc.want[i])
				}
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

// TestSummaryFallback covers the DESCRIPTION -> SYNOPSIS -> NAME fallback
// chain and groff-artifact stripping in summary(). Motivating case is
// `aws s3api`, whose help has no DESCRIPTION body and previously leaked a
// groff page header into the manifest as `S3API()    S3API()`.
func TestSummaryFallback(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "s3api empty description empty name returns empty",
			in: `S3API()                                                                S3API()

NAME
       s3api -

DESCRIPTION
AVAILABLE COMMANDS

       o abort-multipart-upload

                                                                       S3API()
`,
			want: "",
		},
		{
			name: "description present wins over name",
			in: `ROUTE53()                                                            ROUTE53()

NAME
       route53 -

DESCRIPTION
       Amazon Route 53 is a highly available and scalable Domain Name System
       (DNS) web service.
`,
			want: "Amazon Route 53 is a highly available and scalable Domain Name System (DNS) web service.",
		},
		{
			name: "name populated falls back when description empty",
			in: `FOO()                                                                    FOO()

NAME
       foo - does the foo thing really well

DESCRIPTION
SYNOPSIS
`,
			want: "does the foo thing really well",
		},
		{
			name: "synopsis fallback when description missing and name placeholder",
			in: `BAR()                                                                    BAR()

NAME
       bar -

SYNOPSIS
       bar is a command that does bar-y operations.
`,
			want: "bar is a command that does bar-y operations.",
		},
		{
			name: "docutils warnings stripped",
			in: `BAZ()                                                                    BAZ()

<string>:42: (WARNING/2) Inline substitution_reference start-string
NAME
       baz - something useful

DESCRIPTION
       Baz handles the baz workflow end to end.
`,
			want: "Baz handles the baz workflow end to end.",
		},
		{
			name: "non-mannish help uses first sentence",
			in: `Authenticate gh and git with GitHub.

USAGE
  gh auth <command>
`,
			want: "Authenticate gh and git with GitHub.",
		},
		{
			name: "footer page header line stripped",
			in: `S3API()                                                                S3API()

NAME
       s3api -

DESCRIPTION
                                                                       S3API()
`,
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := summary(tc.in)
			if got != tc.want {
				t.Errorf("summary() = %q, want %q", got, tc.want)
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
