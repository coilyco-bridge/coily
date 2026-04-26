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
//
// Side effects:
//
//   - With -capture <dir>, also writes <dir>/<bin>/<path...>/help.txt for each
//     help page fetched, so that tests can run the parser against fixtures
//     instead of the live CLI.
//   - Always writes cmd/subcli-scope/testdata/<bin>.classified.txt: one line
//     per discovered verb, including ones the scope filter rejects, with the
//     verbclass.Label. Diffing this file in `make update-fixtures` is the
//     cheap way to catch a mis-classified mutator before it slips into a
//     pass-through with the wrong policy gate.
//
// The output is consumed by a later codegen step that emits urfave/cli v3
// command trees into pkg/ops/<binary>/generated.go.
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
	"sort"
	"strings"
	"time"

	"github.com/coilysiren/coily/pkg/verbclass"
	"gopkg.in/yaml.v3"
)

// Flag type vocabulary. Lives in one place so the heuristic functions and
// the codegen on the other side of the YAML round-trip agree on spelling.
const (
	flagTypeString      = "string"
	flagTypeBool        = "bool"
	flagTypeInt         = "int"
	flagTypeStringSlice = "stringSlice"
)

// Help section header tokens. Centralized so the per-dialect parsers stay
// in lockstep when an upstream renames a section.
const (
	sectionOptions       = "OPTIONS"
	sectionGlobalOptions = "GLOBAL OPTIONS"
)

type Scope struct {
	Binary string   `yaml:"binary"`
	Mode   string   `yaml:"mode"` // "allow" or "deny"
	Roots  []string `yaml:"roots"`
}

// Flag is one named flag on a command. Type is one of "string", "bool",
// "int", flagTypeStringSlice; the codegen in cmd/gen-passthrough emits a matching
// cli.*Flag for each. Detection is heuristic and falls back to "string".
type Flag struct {
	Name string `yaml:"name"`
	Help string `yaml:"help,omitempty"`
	Type string `yaml:"type,omitempty"`
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

	// Classification snapshot. Walks every verb the discovery saw, including
	// ones the scope filter rejects, and writes the READONLY/MUTATING label
	// per verb. The snapshot lives next to the goldens so weekly fixture
	// regens diff it as part of the same review.
	classified := ClassifyAll(bin, fetch)
	classPath := filepath.Join("cmd", "subcli-scope", "testdata", bin+".classified.txt")
	if err := os.WriteFile(classPath, []byte(renderClassified(bin, classified)), 0o644); err != nil {
		die(err.Error())
	}
	fmt.Fprintf(os.Stderr, "wrote %s: %d verbs\n", classPath, len(classified))
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

// ClassifiedVerb is one row in the per-tool classification snapshot. Sub
// subgroup is the leaf's parent path joined by ".", or "" for top-level
// verbs (kubectl-style). Label is "READONLY" or "MUTATING".
type ClassifiedVerb struct {
	Path  []string
	Label string
}

// ClassifyAll walks the same help tree as Scan but does NOT apply the scope
// filter, so the snapshot covers verbs that are intentionally out of scope.
// That's the point: a verb we filtered out today might be in scope tomorrow,
// and a wrong classification today will then ride along silently.
//
// Only leaves are emitted. Group nodes (`aws ssm`, `kubectl rollout`)
// aren't verbs the user invokes directly, and the prefix heuristic is
// noisy on group names ("backup-gateway" matches the "backup-" prefix
// for example), so including them would clutter the diff with rows that
// aren't actionable.
func ClassifyAll(bin string, fetch HelpFetcher) []ClassifiedVerb {
	topCmds := extractSubcommands(fetch(bin, nil))
	var out []ClassifiedVerb
	seen := map[string]bool{}
	var visit func(path []string, depth int)
	visit = func(path []string, depth int) {
		if depth == 0 {
			return
		}
		key := strings.Join(path, "/")
		if seen[key] {
			return
		}
		seen[key] = true
		help := fetch(bin, path)
		kids := extractSubcommands(help)
		if len(kids) == 0 {
			label := "READONLY"
			if verbclass.Classify(path) != verbclass.Read {
				label = "MUTATING"
			}
			out = append(out, ClassifiedVerb{
				Path:  append([]string{}, path...),
				Label: label,
			})
			return
		}
		for _, k := range kids {
			visit(append(append([]string{}, path...), k), depth-1)
		}
	}
	for _, top := range topCmds {
		visit([]string{top}, 5)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.Join(out[i].Path, " ") < strings.Join(out[j].Path, " ")
	})
	return out
}

// renderClassified formats the snapshot deterministically: one
// `<tool> <path...> LABEL` line per verb, sorted, trailing newline.
func renderClassified(bin string, verbs []ClassifiedVerb) string {
	var sb strings.Builder
	for _, v := range verbs {
		sb.WriteString(bin)
		for _, seg := range v.Path {
			sb.WriteString(" ")
			sb.WriteString(seg)
		}
		sb.WriteString(" ")
		sb.WriteString(v.Label)
		sb.WriteString("\n")
	}
	return sb.String()
}

// liveFetcher invokes `<bin> <path...> help` or `--help` and returns combined
// output with groff overstrike sequences stripped (aws's man-style help).
func liveFetcher(bin string, path []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// aws needs the `help` subcommand to render its man-style page. Other CLIs
	// must use `--help`: gh in particular treats `help` as a positional arg at
	// leaf commands (e.g. `gh search code help` runs a real search for the word
	// "help" and exits 0 with garbage output the parser would then ingest).
	for _, tail := range helpVariants(bin) {
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

func helpVariants(bin string) [][]string {
	if filepath.Base(bin) == "aws" {
		return [][]string{{"help"}, {"--help"}}
	}
	return [][]string{{"--help"}, {"help"}}
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

// extractFlags parses the help text for flag declarations and returns each
// flag with its inferred Type. Three help dialects are handled:
//
//   - aws man-style: each flag appears in OPTIONS / GLOBAL OPTIONS as
//     "--foo (string)" or "--foo (boolean)" with an optional bool partner
//     "--no-foo". The SYNOPSIS block also gives compact shapes like
//     "[--debug | --no-debug]" and "[--region <value>]".
//   - gh-style: a "FLAGS" / "INHERITED FLAGS" / "OPTIONS" block where each
//     line is "  -f, --foo placeholder   description" or "  --foo   desc".
//     A bare flag (no placeholder) is bool. A "key=value" or "key:value"
//     placeholder paired with description text mentioning "Pass one or more"
//     or "Add" of the flag is stringSlice.
//   - kubectl-style: "    --foo=default:" lines in an Options block.
//     "=false"/"=true" → bool, "=[]" → stringSlice, numeric → int.
//
// Default Type when the heuristic doesn't fire is "string". Order matches
// the order flags appear in the help text.
func extractFlags(help string) []Flag {
	flags := flagsFromAWSStyle(help)
	if len(flags) == 0 {
		flags = flagsFromGHStyle(help)
	}
	if len(flags) == 0 {
		flags = flagsFromKubectlStyle(help)
	}
	if len(flags) == 0 {
		// Last-resort fallback: the old loose regex, which catches at least
		// the flag name even from messy output. Type stays empty (= string).
		flags = flagsFromGenericRegex(help)
	}
	return flags
}

// flagsFromAWSStyle picks up aws's man-style OPTIONS and GLOBAL OPTIONS
// sections. Each option begins with a line that starts at 7 spaces of
// indentation and contains "--foo" plus a "(type)" annotation.
//
// Returns the flag list in declaration order. Returns nil if nothing aws-
// shaped was found, so callers can fall through to other parsers.
var awsOptionRE = regexp.MustCompile(`^\s+(--[a-z][a-zA-Z0-9-]*(?:\s*\|\s*--[a-z][a-zA-Z0-9-]*)?)\s*\(([a-z]+)\)`)

func flagsFromAWSStyle(help string) []Flag {
	if !strings.Contains(help, "\n"+sectionOptions+"\n") && !strings.Contains(help, "\n"+sectionGlobalOptions+"\n") {
		return nil
	}
	scanner := bufio.NewScanner(strings.NewReader(help))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	seen := map[string]bool{}
	var out []Flag
	inOptions := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if isAWSOptionsHeader(trimmed) {
			inOptions = true
			continue
		}
		if inOptions && isAllCapsHeader(trimmed) && !isAWSOptionsHeader(trimmed) {
			inOptions = false
		}
		if !inOptions {
			continue
		}
		flag, ok := parseAWSOptionLine(line, seen)
		if !ok {
			continue
		}
		out = append(out, flag)
	}
	return out
}

func isAWSOptionsHeader(trimmed string) bool {
	return trimmed == sectionOptions || trimmed == sectionGlobalOptions
}

// parseAWSOptionLine matches a single OPTIONS line and returns the
// canonicalized Flag plus whether the line was a valid declaration. seen is
// updated in place so callers can dedupe.
func parseAWSOptionLine(line string, seen map[string]bool) (Flag, bool) {
	m := awsOptionRE.FindStringSubmatch(line)
	if m == nil {
		return Flag{}, false
	}
	// m[1] may be "--foo | --no-foo" - take the first.
	flagDecl := m[1]
	typeHint := m[2]
	first := flagDecl
	if i := strings.Index(flagDecl, "|"); i >= 0 {
		first = strings.TrimSpace(flagDecl[:i])
	}
	first = strings.TrimSpace(first)
	if seen[first] {
		return Flag{}, false
	}
	seen[first] = true
	return Flag{Name: first, Type: awsTypeFromHint(typeHint)}, true
}

// awsTypeFromHint maps aws's annotation token to our Type vocabulary. The
// annotations seen in the wild: string, boolean, integer, long, double,
// list, map, structure, blob, timestamp. Lists become stringSlice; numbers
// become int; booleans become bool. Everything else collapses to string,
// because the underlying CLI accepts a JSON literal for those.
func awsTypeFromHint(h string) string {
	switch h {
	case "boolean":
		return flagTypeBool
	case "integer", "long":
		return flagTypeInt
	case "list":
		return flagTypeStringSlice
	default:
		return flagTypeString
	}
}

// flagsFromGHStyle parses gh-flavored help. The block(s) look like:
//
//	FLAGS
//	  -F, --field key=value       Add a typed parameter in key=value format
//	  -X, --method string         The HTTP method for the request (default "GET")
//	      --paginate              Make additional HTTP requests to fetch all pages of results
//
// Each line in a FLAGS / INHERITED FLAGS / OPTIONS section is parsed for
// short alias, long name, optional placeholder, and description. The
// placeholder + description text drive Type detection.
// Placeholder leading char includes "[" so bracketed shapes like
// "[HOST/]OWNER/REPO" (gh's --repo, INHERITED FLAGS) parse cleanly. Without
// it the whole line failed and the inherited flag was silently dropped.
var ghFlagLineRE = regexp.MustCompile(`^\s{2,6}(?:-[a-zA-Z],\s+)?(--[a-z][a-zA-Z0-9-]*)(?:\s+([A-Za-z\[][A-Za-z0-9_:=\[\]/.,-]*))?(\s{2,}.*)?$`)

func flagsFromGHStyle(help string) []Flag {
	scanner := bufio.NewScanner(strings.NewReader(help))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	seen := map[string]bool{}
	var out []Flag
	inFlags := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if isGHFlagsHeader(trimmed) {
			inFlags = true
			continue
		}
		if inFlags && endsGHFlagsBlock(line, trimmed) {
			inFlags = false
			continue
		}
		if !inFlags {
			continue
		}
		flag, ok := parseGHFlagLine(line, seen)
		if !ok {
			continue
		}
		out = append(out, flag)
	}
	return out
}

func isGHFlagsHeader(trimmed string) bool {
	upper := strings.ToUpper(trimmed)
	return upper == "FLAGS" || upper == "INHERITED FLAGS" || upper == sectionOptions || upper == sectionOptions+":"
}

// endsGHFlagsBlock detects an ALLCAPS section header at column 0, which is
// gh's signal that the FLAGS / INHERITED FLAGS block has ended.
func endsGHFlagsBlock(line, trimmed string) bool {
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(line, " ") {
		return false
	}
	if line != strings.ToUpper(line) {
		return false
	}
	return isAllCapsHeader(trimmed)
}

func parseGHFlagLine(line string, seen map[string]bool) (Flag, bool) {
	m := ghFlagLineRE.FindStringSubmatch(line)
	if m == nil {
		return Flag{}, false
	}
	name := m[1]
	placeholder := m[2]
	desc := strings.TrimSpace(m[3])
	if seen[name] {
		return Flag{}, false
	}
	seen[name] = true
	return Flag{Name: name, Type: ghType(placeholder, desc)}, true
}

// ghType infers the flag type from gh's placeholder + description. Empty
// placeholder means a boolean switch. A "key=value" or "key:value"
// placeholder is the strongest stringSlice tell because gh uses it for
// repeatable -F/--field, -H/--header, -f/--raw-field. "int"/"count"
// placeholders are int. Everything else is string.
func ghType(placeholder, desc string) string {
	if placeholder == "" {
		return flagTypeBool
	}
	if strings.Contains(placeholder, "=") || strings.Contains(placeholder, ":") {
		return flagTypeStringSlice
	}
	low := strings.ToLower(placeholder)
	if low == "int" || low == "count" || low == "n" || low == "number" {
		return flagTypeInt
	}
	// Description-text repeatable hints. These are the ones we've seen in
	// the wild for gh and similar.
	dl := strings.ToLower(desc)
	for _, hint := range []string{
		"can be specified multiple times",
		"can be repeated",
		"pass one or more",
		"may be specified more than once",
	} {
		if strings.Contains(dl, hint) {
			return flagTypeStringSlice
		}
	}
	return flagTypeString
}

// flagsFromKubectlStyle parses kubectl-flavored help. The block looks like:
//
//	Options:
//	    --all=false:
//		Select all resources in the namespace of the specified resource types.
//	    -f, --filename=[]:
//		The files that contain the configurations to apply.
//
// The default value after `=` is the type tell.
var kubectlFlagLineRE = regexp.MustCompile(`^\s{2,8}(?:-[a-zA-Z],\s+)?(--[a-z][a-zA-Z0-9-]*)=(.*?):\s*$`)

func flagsFromKubectlStyle(help string) []Flag {
	if !strings.Contains(help, "\nOptions:\n") {
		return nil
	}
	scanner := bufio.NewScanner(strings.NewReader(help))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	seen := map[string]bool{}
	var out []Flag
	inOpts := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "Options:" {
			inOpts = true
			continue
		}
		// kubectl ends Options with a "Usage:" header.
		if inOpts && trimmed == "Usage:" {
			inOpts = false
			continue
		}
		if !inOpts {
			continue
		}
		m := kubectlFlagLineRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		def := strings.TrimSpace(m[2])
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, Flag{Name: name, Type: kubectlType(def)})
	}
	return out
}

// kubectlType infers Type from the printed default. Defaults like "false",
// "true" → bool. "[]" → stringSlice. A bare integer → int. Everything else
// (string default, duration like "0s", quoted string) → string.
func kubectlType(def string) string {
	switch def {
	case "false", "true":
		return flagTypeBool
	case "[]":
		return flagTypeStringSlice
	}
	// Strip surrounding quotes.
	if len(def) >= 2 && (def[0] == '\'' || def[0] == '"') && def[len(def)-1] == def[0] {
		return flagTypeString
	}
	if isAllDigits(strings.TrimPrefix(def, "-")) {
		return flagTypeInt
	}
	return flagTypeString
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// flagsFromGenericRegex is the legacy loose parser. Used only when the three
// shaped parsers all returned nothing. Type stays empty so the codegen
// defaults to StringFlag.
func flagsFromGenericRegex(help string) []Flag {
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

// commandSectionHeaderRE matches every shape of "commands" section header
// across our supported CLIs. kubectl groups commands under headers like
// "Basic Commands (Beginner):", "Deploy Commands:", "Other Commands:".
// gh and aws use plain "COMMANDS:" / "SUBCOMMANDS:" / "SERVICES:" / "TOPICS:".
// Allow an optional parenthetical qualifier after the keyword and a trailing
// colon, since both vary across sections and tools.
var commandSectionHeaderRE = regexp.MustCompile(`^(?:[A-Z0-9-]+\s+)*(?:COMMANDS|SUBCOMMANDS|SERVICES|TOPICS)(?:\s*\([^)]*\))?:?$`)

func isCommandSectionHeader(upper string) bool {
	return commandSectionHeaderRE.MatchString(upper)
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
	// Man-page page header/footer like "ROUTE53()                        ROUTE53()".
	// Present as line 1 (and also the final line) of aws help output after
	// groff overstrikes are stripped. Matches an all-line occurrence so it can
	// be removed as noise wherever it appears.
	manPageHeaderRE = regexp.MustCompile(`^[A-Z][A-Z0-9-]*\(\)`)
	// Groff page header and footer lines. The header is two copies of
	// `NAME()` separated by whitespace; the footer is the same token indented
	// on its own. Stripping both removes garbage like `S3API()    S3API()` and
	// the trailing `S3API()` line that otherwise leak into parsed output.
	manPageHeaderLineRE = regexp.MustCompile(`(?m)^\s*[A-Z][A-Z0-9-]*\(\)\s+[A-Z][A-Z0-9-]*\(\)\s*$\n?`)
	manPageFooterLineRE = regexp.MustCompile(`(?m)^\s*[A-Z][A-Z0-9-]*\(\)\s*$\n?`)
	// docutils parser warnings that aws's help sometimes emits inline,
	// e.g. "<string>:272: (WARNING/2) Inline substitution_reference ...".
	docutilsWarningRE = regexp.MustCompile(`(?m)^<string>:\d+: \([A-Z]+/\d+\).*$\n?`)
)

// summary returns a clean one-sentence description of the command the help
// text is for. Handles three shapes.
//
//   - aws man-style: line 1 is `COMMAND()   COMMAND()`. Pull from the
//     DESCRIPTION section body, falling back to SYNOPSIS then NAME when
//     DESCRIPTION is missing or empty (aws s3api is the motivating case).
//   - gh/kubectl: line 1 IS the description. Take the first sentence.
//   - empty: return empty.
//
// Also strips docutils parser warnings and groff page header/footer lines
// that leak into aws help text.
func summary(help string) string {
	help = docutilsWarningRE.ReplaceAllString(help, "")
	// Detect mannish shape BEFORE stripping groff header/footer lines so we
	// don't false-positive on kubectl fixtures that happen to start with a
	// bare "NAME" line (broken captures).
	wasMannish := manPageHeaderRE.MatchString(firstLine(strings.TrimSpace(help)))
	help = manPageHeaderLineRE.ReplaceAllString(help, "")
	help = manPageFooterLineRE.ReplaceAllString(help, "")
	help = strings.TrimSpace(help)
	if help == "" {
		return ""
	}

	if wasMannish {
		// Try DESCRIPTION first, then SYNOPSIS, then NAME. Any empty or
		// placeholder-only section is skipped so we don't return "foo -".
		for _, section := range []string{"DESCRIPTION", "SYNOPSIS", "NAME"} {
			if d := mannishSection(help, section); d != "" {
				return d
			}
		}
		// Mannish but every section is empty. Don't fall through to
		// firstSentence, which would return the groff header line.
		return ""
	}
	return firstSentence(help)
}

// mannishSection walks a man-style help page and returns the first sentence
// of the requested section's body, or "" if the section is missing, empty,
// or contains only a placeholder like "s3api -".
func mannishSection(help, header string) string {
	scanner := bufio.NewScanner(strings.NewReader(help))
	inSection := false
	var buf strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if !inSection {
			if trimmed == header {
				inSection = true
			}
			continue
		}
		// We're inside the section. A blank line after we've collected text
		// ends the first paragraph, which is where the summary sentence lives.
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
	body := strings.TrimSpace(buf.String())
	// NAME bodies look like "s3api -" or "s3api - manage S3 at the API level".
	// Strip the leading "name -" prefix so we don't emit a dangling dash.
	// When the body is just "name" or "name -", treat it as an empty
	// placeholder (aws s3api shape) so the caller can fall through to other
	// sections or return empty.
	if header == "NAME" {
		if i := strings.Index(body, " - "); i >= 0 {
			body = strings.TrimSpace(body[i+3:])
		} else if strings.HasSuffix(body, " -") {
			body = ""
		} else if !strings.ContainsAny(body, " \t") {
			// Single token, no description. Placeholder.
			body = ""
		}
	}
	return firstSentence(body)
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
