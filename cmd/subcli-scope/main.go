// subcli-scope walks a target CLI's help output and emits a structured manifest
// of its commands and flags, filtered by an allow/deny scope yaml.
//
// Usage:
//
//	go run ./cmd/subcli-scope <binary>
//	go run ./cmd/subcli-scope -capture testdata/fixtures <binary>   # record help outputs
//
// Reads  configs/scopes/<binary>.yaml
// Writes configs/commands/<binary>.yaml
// With -capture <dir>, also writes <dir>/<bin>/<path...>/help.txt for each
// help page fetched, so that tests can run the parser against fixtures instead
// of the live CLI.
//
// The output is consumed by a later codegen step (not yet written) that emits
// urfave/cli v3 command trees into pkg/ops/<binary>/generated.go.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Scope struct {
	Binary string   `yaml:"binary"`
	Mode   string   `yaml:"mode"` // "allow" or "deny"
	Roots  []string `yaml:"roots"`
}

type Flag struct {
	Name string `yaml:"name"`
	Help string `yaml:"help,omitempty"`
}

type Command struct {
	Path  []string `yaml:"path"`
	Help  string   `yaml:"help,omitempty"`
	Flags []Flag   `yaml:"flags,omitempty"`
	// Subcommands with their own leaves; populated during scan.
	Children []string `yaml:"children,omitempty"`
}

type Manifest struct {
	Binary     string    `yaml:"binary"`
	BinVersion string    `yaml:"bin_version,omitempty"`
	ScannedAt  string    `yaml:"scanned_at"`
	Commands   []Command `yaml:"commands"`
}

// HelpFetcher returns the help-text output for a given subcommand path of a
// binary. Concrete impls: liveFetcher (shells out) and fixtureFetcher (reads
// from testdata/). This indirection is what lets tests run the parser against
// recorded fixtures instead of the installed CLI.
type HelpFetcher func(bin string, path []string) string

func main() {
	var captureDir string
	flag.StringVar(&captureDir, "capture", "", "directory to write captured help text into (for tests)")
	flag.Parse()

	if flag.NArg() != 1 {
		die("usage: subcli-scope [-capture DIR] <binary>")
	}
	bin := flag.Arg(0)

	scope, err := loadScope(bin)
	if err != nil {
		die(err.Error())
	}

	fetch := liveFetcher
	if captureDir != "" {
		fetch = capturingFetcher(captureDir, liveFetcher)
	}

	m := Manifest{
		Binary:     bin,
		BinVersion: binVersion(bin),
		ScannedAt:  time.Now().UTC().Format(time.RFC3339),
		Commands:   Scan(bin, scope, fetch),
	}

	out, err := yaml.Marshal(&m)
	if err != nil {
		die(err.Error())
	}
	outPath := filepath.Join("configs", "commands", bin+".yaml")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		die(err.Error())
	}
	if err := os.WriteFile(outPath, out, 0o644); err != nil {
		die(err.Error())
	}
	fmt.Fprintf(os.Stderr, "wrote %s: %d command leaves\n", outPath, len(m.Commands))
}

func die(msg string) {
	fmt.Fprintln(os.Stderr, "subcli-scope: "+msg)
	os.Exit(1)
}

func loadScope(bin string) (*Scope, error) {
	path := filepath.Join("configs", "scopes", bin+".yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read scope %s: %w", path, err)
	}
	var s Scope
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parse scope %s: %w", path, err)
	}
	if s.Binary != bin {
		return nil, fmt.Errorf("scope file declares binary %q but called with %q", s.Binary, bin)
	}
	if s.Mode != "allow" && s.Mode != "deny" {
		return nil, fmt.Errorf("scope mode must be 'allow' or 'deny', got %q", s.Mode)
	}
	return &s, nil
}

func binVersion(bin string) string {
	for _, args := range [][]string{{"--version"}, {"version"}, {"-v"}} {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, bin, args...).CombinedOutput()
		if err == nil {
			return strings.TrimSpace(firstLine(string(out)))
		}
	}
	return ""
}

// Scan walks the CLI's help output recursively, respecting scope. Pure
// function of (bin, scope, fetch), so tests can inject a fixture fetcher.
func Scan(bin string, scope *Scope, fetch HelpFetcher) []Command {
	topCmds := extractSubcommands(fetch(bin, nil))
	var leaves []Command

	for _, top := range topCmds {
		inScope := slices.Contains(scope.Roots, top)
		if scope.Mode == "allow" && !inScope {
			continue
		}
		if scope.Mode == "deny" && inScope {
			continue
		}
		walk(bin, []string{top}, &leaves, 4, fetch) // max depth 4
	}
	return leaves
}

// walk recursively enumerates subcommands under `path`, appending command leaves.
func walk(bin string, path []string, leaves *[]Command, depth int, fetch HelpFetcher) {
	if depth == 0 {
		return
	}
	help := fetch(bin, path)
	kids := extractSubcommands(help)
	flags := extractFlags(help)

	if len(kids) == 0 {
		*leaves = append(*leaves, Command{
			Path:  append([]string{}, path...),
			Help:  summary(help),
			Flags: flags,
		})
		return
	}

	// Internal node. Emit so codegen can create the grouping.
	*leaves = append(*leaves, Command{
		Path:     append([]string{}, path...),
		Help:     summary(help),
		Children: kids,
	})
	for _, k := range kids {
		walk(bin, append(append([]string{}, path...), k), leaves, depth-1, fetch)
	}
}

// liveFetcher invokes `<bin> <path...> help` or `--help` and returns combined
// output with groff overstrike sequences stripped (aws's man-style help).
func liveFetcher(bin string, path []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// Try `help` subcommand first (aws style), fall back to --help (gh/kubectl).
	for _, tail := range [][]string{{"help"}, {"--help"}} {
		args := append(append([]string{}, path...), tail...)
		cmd := exec.CommandContext(ctx, bin, args...)
		// Defeat pagers. aws CLI pipes help through less by default.
		cmd.Env = append(os.Environ(),
			"PAGER=cat",
			"MANPAGER=cat",
			"AWS_PAGER=",
			"LESS=",
			"TERM=dumb",
		)
		out, err := cmd.CombinedOutput()
		if err == nil && len(out) > 0 {
			return stripOverstrike(string(out))
		}
	}
	return ""
}

// capturingFetcher wraps an inner fetcher and also writes each help text to
// <dir>/<bin>/<path-joined-by-slashes>/help.txt. Used by `-capture` to record
// test fixtures from a live run.
func capturingFetcher(dir string, inner HelpFetcher) HelpFetcher {
	return func(bin string, path []string) string {
		help := inner(bin, path)
		outDir := filepath.Join(append([]string{dir, bin}, path...)...)
		_ = os.MkdirAll(outDir, 0o755)
		_ = os.WriteFile(filepath.Join(outDir, "help.txt"), []byte(help), 0o644)
		return help
	}
}

// fixtureFetcher reads help text from <dir>/<bin>/<path...>/help.txt. Missing
// files return the empty string (treated as "no subcommands", same as live).
func fixtureFetcher(dir string) HelpFetcher {
	return func(bin string, path []string) string {
		p := filepath.Join(append([]string{dir, bin}, path...)...)
		p = filepath.Join(p, "help.txt")
		b, err := os.ReadFile(p)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

// stripOverstrike removes groff-style backspace overstrikes: X\bX becomes X.
// Equivalent to `col -b`.
func stripOverstrike(s string) string {
	runes := []rune(s)
	out := make([]rune, 0, len(runes))
	for _, r := range runes {
		if r == '\b' {
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

var (
	// Matches command lines across help formats:
	//   kubectl: "  alpha         Commands for features in alpha"
	//   gh:      "  auth:        Authenticate gh and git with GitHub"
	//   aws:     "   o accessanalyzer"        (see cmdLineREAWS below)
	// The optional colon handles gh. The required 2+ spaces or EOL after
	// separates the name from the description.
	cmdLineRE    = regexp.MustCompile(`(?m)^\s{2,6}([a-z][a-z0-9-]{1,40}):?(?:\s{2,}|$)`)
	cmdLineREAWS = regexp.MustCompile(`(?m)^\s+o\s+([a-z][a-z0-9-]{1,40})\s*$`)
	// Flag lines like "  --foo string   description" or "  -f, --foo".
	flagLineRE = regexp.MustCompile(`(?m)^\s{2,6}(?:-[a-zA-Z],\s*)?(--[a-z][a-zA-Z0-9-]+)`)
)

func extractSubcommands(help string) []string {
	secs := sections(help)
	seen := map[string]bool{}
	var out []string
	for _, sec := range secs {
		for _, m := range cmdLineRE.FindAllStringSubmatch(sec, -1) {
			name := m[1]
			if seen[name] || isNoisyCmdName(name) {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
		for _, m := range cmdLineREAWS.FindAllStringSubmatch(sec, -1) {
			name := m[1]
			if seen[name] || isNoisyCmdName(name) {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

func extractFlags(help string) []Flag {
	seen := map[string]bool{}
	var out []Flag
	for _, m := range flagLineRE.FindAllStringSubmatch(help, -1) {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, Flag{Name: name})
	}
	return out
}

// sections extracts every "COMMAND(S)" / "SUBCOMMANDS" / "SERVICES" / "TOPICS"
// section body from help text. Each returned string is the body (lines) of
// one such section, with headers stripped.
func sections(help string) []string {
	scanner := bufio.NewScanner(strings.NewReader(help))
	var all []string
	var buf strings.Builder
	inSection := false
	flush := func() {
		if inSection && buf.Len() > 0 {
			all = append(all, buf.String())
		}
		buf.Reset()
		inSection = false
	}
	for scanner.Scan() {
		line := scanner.Text()
		upper := strings.ToUpper(strings.TrimSpace(line))
		if isCommandSectionHeader(upper) {
			flush()
			inSection = true
			continue
		}
		if inSection && looksLikeHeader(line) {
			flush()
			continue
		}
		if inSection {
			buf.WriteString(line + "\n")
		}
	}
	flush()
	return all
}

func isCommandSectionHeader(upper string) bool {
	for _, suffix := range []string{"COMMANDS", "COMMANDS:", "SUBCOMMANDS", "SUBCOMMANDS:", "SERVICES", "SERVICES:", "TOPICS", "TOPICS:"} {
		if strings.HasSuffix(upper, suffix) {
			return true
		}
	}
	return false
}

func looksLikeHeader(line string) bool {
	t := strings.TrimSpace(line)
	if t == "" {
		return false
	}
	if strings.HasSuffix(t, ":") {
		return true
	}
	if t != strings.ToUpper(t) || len(t) <= 2 || strings.HasPrefix(t, "-") {
		return false
	}
	alpha := 0
	for _, r := range t {
		if r >= 'A' && r <= 'Z' {
			alpha++
		}
	}
	return alpha >= 3
}

func isNoisyCmdName(name string) bool {
	switch name {
	case "help", "version", "completion":
		return true
	}
	return false
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

var (
	// Man-page page header like "ROUTE53()                        ROUTE53()".
	// Present as line 1 of aws help output after groff overstrikes are stripped.
	manPageHeaderRE = regexp.MustCompile(`^[A-Z][A-Z0-9-]*\(\)`)
	// docutils parser warnings that aws's help sometimes emits inline,
	// e.g. "<string>:272: (WARNING/2) Inline substitution_reference ...".
	docutilsWarningRE = regexp.MustCompile(`(?m)^<string>:\d+: \([A-Z]+/\d+\).*$\n?`)
)

// summary returns a clean one-sentence description of the command the help
// text is for. Handles three shapes.
//
//   - aws man-style: line 1 is `COMMAND()   COMMAND()`. Pull from the
//     DESCRIPTION section body, skipping NAME and other preamble.
//   - gh/kubectl: line 1 IS the description. Take the first sentence.
//   - empty: return empty.
//
// Also strips docutils parser warnings that leak into aws help text.
func summary(help string) string {
	help = docutilsWarningRE.ReplaceAllString(help, "")
	help = strings.TrimSpace(help)
	if help == "" {
		return ""
	}

	if manPageHeaderRE.MatchString(firstLine(help)) {
		if d := mannishDescription(help); d != "" {
			return d
		}
	}
	return firstSentence(help)
}

// mannishDescription walks the text of a man-style help page and returns the
// first sentence of its DESCRIPTION section body, or "" if no DESCRIPTION
// block is present.
func mannishDescription(help string) string {
	scanner := bufio.NewScanner(strings.NewReader(help))
	inDescription := false
	var buf strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if !inDescription {
			if trimmed == "DESCRIPTION" {
				inDescription = true
			}
			continue
		}
		// We're in DESCRIPTION. A blank line after we've collected text ends
		// the first paragraph, which is where the summary sentence lives.
		if trimmed == "" {
			if buf.Len() > 0 {
				break
			}
			continue
		}
		// Another ALLCAPS section header (SYNOPSIS, OPTIONS, NAME) ends it.
		if isAllCapsHeader(trimmed) {
			break
		}
		if buf.Len() > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(trimmed)
	}
	return firstSentence(buf.String())
}

// isAllCapsHeader matches section headers like "DESCRIPTION", "SYNOPSIS",
// "AVAILABLE COMMANDS", "GLOBAL OPTIONS". Must be >=3 chars, uppercase only.
func isAllCapsHeader(s string) bool {
	if len(s) < 3 {
		return false
	}
	if s != strings.ToUpper(s) {
		return false
	}
	for _, r := range s {
		isUpper := r >= 'A' && r <= 'Z'
		isDigit := r >= '0' && r <= '9'
		isPunct := r == ' ' || r == '-' || r == '/'
		if !isUpper && !isDigit && !isPunct {
			return false
		}
	}
	return true
}

func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Sentence end on '.' followed by space/newline/EOS.
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			if i+1 >= len(s) || s[i+1] == ' ' || s[i+1] == '\n' {
				return strings.TrimSpace(s[:i+1])
			}
		}
		if s[i] == '\n' {
			return strings.TrimSpace(s[:i])
		}
	}
	if len(s) > 200 {
		return s[:200]
	}
	return s
}
