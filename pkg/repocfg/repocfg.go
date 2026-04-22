// Package repocfg loads a per-repo command allowlist from a coily.yaml file
// discovered by walking up from the current working directory. Each command
// in the file becomes a top-level coily verb (coily test, coily lint, etc.)
// that expands to a pre-declared argv.
//
// Repo commands share the same security boundary as every other coily verb.
// Every argv token (both the tokens from coily.yaml and any user-supplied
// extras appended at invocation) is run through policy.ValidateArg, which
// rejects shell metacharacters. There is no "it's just a dev script" carve-
// out. If a repo command needs a shell pipeline, the repo author's answer is
// a wrapper script, not an escape hatch in coily.
//
// Unlike the embedded tool config, coily.yaml is loaded from disk at runtime.
// The blast radius is bounded by the same policy checks, and the file is
// expected to be checked into git where a human reviews diffs. Per-repo
// dev tools change too often to warrant a rebuild+install cycle.
package repocfg

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coilysiren/coily/pkg/policy"
	"gopkg.in/yaml.v3"
)

// Filename is the name repocfg looks for during discovery.
const Filename = "coily.yaml"

// EnvOverride, when set, is treated as the absolute path to the repo config
// and skips directory walking. Primarily for tests and advanced users.
const EnvOverride = "COILY_REPO_CONFIG"

// Command is one parsed and validated entry from the commands: map.
type Command struct {
	// Name is the key from the yaml map, e.g. "test".
	Name string
	// Description is the optional human-readable blurb shown in help/--list.
	Description string
	// Argv is the command split into tokens. argv[0] is the binary name as
	// resolved via $PATH at exec time. Every token has already been run
	// through policy.ValidateArg.
	Argv []string
}

// Config is the result of a successful Load.
type Config struct {
	// Path is the absolute path to the coily.yaml that produced this Config.
	Path string
	// Commands are sorted by Name. Safe to iterate directly for help output.
	Commands []Command
}

// ErrNoConfig is returned by LoadDefault when no coily.yaml is found in the
// cwd ancestry. Callers treat this as "no repo commands to register."
var ErrNoConfig = errors.New("repocfg: no coily.yaml found")

// Discover walks up from start looking for Filename. Returns the absolute
// path on success, "" and ErrNoConfig when no file exists in the ancestry.
func Discover(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("repocfg: abs %s: %w", start, err)
	}
	for {
		candidate := filepath.Join(dir, Filename)
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("repocfg: stat %s: %w", candidate, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNoConfig
		}
		dir = parent
	}
}

// LoadDefault resolves the config path from $COILY_REPO_CONFIG or by walking
// up from the current working directory, then parses it. Returns nil,
// ErrNoConfig when no config is found. All other errors are parsing or
// validation failures and are surfaced as-is.
func LoadDefault() (*Config, error) {
	if override := os.Getenv(EnvOverride); override != "" {
		return Load(override)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("repocfg: getwd: %w", err)
	}
	path, err := Discover(cwd)
	if err != nil {
		return nil, err
	}
	return Load(path)
}

// Load parses the yaml at path. Every command is validated against
// policy.ValidateArg. A single bad token fails the whole load.
func Load(path string) (*Config, error) {
	path = filepath.Clean(path)
	b, err := os.ReadFile(path) // #nosec G304 -- caller-controlled config path is the intended input

	if err != nil {
		return nil, fmt.Errorf("repocfg: read %s: %w", path, err)
	}
	var raw struct {
		Commands map[string]yaml.Node `yaml:"commands"`
	}
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("repocfg: parse %s: %w", path, err)
	}

	cfg := &Config{Path: path}
	for name, node := range raw.Commands {
		cmd, err := decodeCommand(name, node)
		if err != nil {
			return nil, fmt.Errorf("repocfg: %s: command %q: %w", path, name, err)
		}
		cfg.Commands = append(cfg.Commands, cmd)
	}
	sort.Slice(cfg.Commands, func(i, j int) bool {
		return cfg.Commands[i].Name < cfg.Commands[j].Name
	})
	return cfg, nil
}

func decodeCommand(name string, node yaml.Node) (Command, error) {
	if err := validateName(name); err != nil {
		return Command{}, err
	}
	var runStr, desc string
	switch node.Kind {
	case yaml.ScalarNode:
		if err := node.Decode(&runStr); err != nil {
			return Command{}, fmt.Errorf("decode scalar: %w", err)
		}
	case yaml.MappingNode:
		var obj struct {
			Run         string `yaml:"run"`
			Description string `yaml:"description"`
		}
		if err := node.Decode(&obj); err != nil {
			return Command{}, fmt.Errorf("decode mapping: %w", err)
		}
		runStr = obj.Run
		desc = obj.Description
	case yaml.DocumentNode, yaml.SequenceNode, yaml.AliasNode:
		return Command{}, fmt.Errorf("must be a string or a {run, description} mapping")
	default:
		return Command{}, fmt.Errorf("must be a string or a {run, description} mapping")
	}
	runStr = strings.TrimSpace(runStr)
	if runStr == "" {
		return Command{}, errors.New("run is empty")
	}
	argv := strings.Fields(runStr)
	if len(argv) == 0 {
		return Command{}, errors.New("run parsed to zero tokens")
	}
	for i, tok := range argv {
		if err := policy.ValidateArg(fmt.Sprintf("argv[%d]", i), tok); err != nil {
			return Command{}, err
		}
	}
	return Command{Name: name, Description: desc, Argv: argv}, nil
}

// validateName rejects command names that would confuse cli parsing or help
// output. Keep it tight: lowercase letters, digits, and single-dashes.
func validateName(name string) error {
	if name == "" {
		return errors.New("name is empty")
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("name %q cannot start or end with '-'", name)
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return fmt.Errorf("name %q contains illegal character %q (allowed: a-z, 0-9, -)", name, r)
		}
	}
	return nil
}
