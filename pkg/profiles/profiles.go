// Package profiles loads the per-host lockdown profile registry from
// ~/.coily/coily.yaml and resolves named profiles to cli-guard/profile
// Coordinates. The override file is the only source of non-strictest
// tiers: if it is absent, every profile resolves to Strictest(). The
// embedded default.yaml is a template the operator copies into place
// via `coily lockdown init-config`, never silently active.
//
// Phase 3 of coilysiren/coily#150. No gate behavior yet; phase 4 wires
// the resolved Coordinate into the lockdown evaluator.
package profiles

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/coilysiren/cli-guard/profile"
	"gopkg.in/yaml.v3"
)

//go:embed default.yaml
var DefaultYAML []byte

// File is the on-disk shape of ~/.coily/coily.yaml. One field today;
// future schema additions land alongside without breaking the loader.
type File struct {
	Profiles map[string]rawCoordinate `yaml:"profiles"`
}

// rawCoordinate carries the four-tier declaration verbatim from YAML.
// Validation (axis presence, tier membership) runs on this; the
// validated form is profile.Coordinate.
type rawCoordinate struct {
	DataSecurity    string `yaml:"data_security"`
	BlastRadius     string `yaml:"blast_radius"`
	NetworkEgress   string `yaml:"network_egress"`
	FilesystemReach string `yaml:"filesystem_reach"`
}

// Source records why a Coordinate resolved the way it did. The session
// show command surfaces this so the operator can see at a glance
// whether the active tiers came from their override or the
// deny-everything fallback.
type Source string

const (
	// SourceOverride means the named profile was found in the on-disk
	// override file and its tiers are in effect.
	SourceOverride Source = "override"

	// SourceMissingFile means ~/.coily/coily.yaml was absent. Every
	// axis falls back to Strictest().
	SourceMissingFile Source = "missing_file"

	// SourceUnknownProfile means the override file exists and parsed,
	// but the requested profile name was not declared in it. Every
	// axis falls back to Strictest().
	SourceUnknownProfile Source = "unknown_profile"

	// SourceUnset means no profile name was requested (the session
	// sentinel was absent). Every axis falls back to Strictest().
	SourceUnset Source = "unset"
)

// Resolution is the output of Resolve: the resolved Coordinate, the
// source that produced it, and a human-readable note suitable for the
// show command's output.
type Resolution struct {
	Coord  profile.Coordinate
	Source Source
	Note   string
}

// OverridePath returns ~/.coily/coily.yaml. Caller stat()s it; the
// loader treats os.ErrNotExist as the deny-everything fallback signal.
func OverridePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("profiles: resolve home: %w", err)
	}
	return filepath.Join(home, ".coily", "coily.yaml"), nil
}

// LoadOverride reads and validates ~/.coily/coily.yaml. Returns
// (nil, os.ErrNotExist) when the file is absent so callers can fall
// back to Strictest(). Other errors are loud and kind-tagged.
func LoadOverride() (map[string]profile.Coordinate, error) {
	path, err := OverridePath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path) //nolint:gosec // path is under $HOME/.coily, fixed name
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return nil, fmt.Errorf("profiles: read %s: %w", path, err)
	}
	return ParseAndValidate(b, path)
}

// ParseAndValidate decodes a YAML body and validates every profile.
// Exposed for tests and for `coily lockdown init-config` to check the
// embedded template before writing it.
func ParseAndValidate(body []byte, sourceDesc string) (map[string]profile.Coordinate, error) {
	var f File
	if err := yaml.Unmarshal(body, &f); err != nil {
		return nil, fmt.Errorf("profiles: parse %s: %w", sourceDesc, err)
	}
	if len(f.Profiles) == 0 {
		return nil, fmt.Errorf("profiles: %s declares no profiles", sourceDesc)
	}
	out := make(map[string]profile.Coordinate, len(f.Profiles))
	for name, raw := range f.Profiles {
		if err := validateProfileName(name); err != nil {
			return nil, fmt.Errorf("profiles: %s: %w", sourceDesc, err)
		}
		coord, err := validateRaw(name, raw)
		if err != nil {
			return nil, fmt.Errorf("profiles: %s: %w", sourceDesc, err)
		}
		out[name] = coord
	}
	return out, nil
}

// Resolve returns the Coordinate for the given profile name. An empty
// name resolves to Strictest with Source=Unset so the show command can
// distinguish "no sentinel" from "unknown profile". Errors propagate
// from LoadOverride only when the override file is malformed; absence
// of the file is not an error, it is the deny-everything fallback.
func Resolve(profileName string) (Resolution, error) {
	if profileName == "" {
		return strictest(SourceUnset, "no profile selected for this session"), nil
	}
	loaded, err := LoadOverride()
	if errors.Is(err, os.ErrNotExist) {
		path, _ := OverridePath()
		return strictest(SourceMissingFile,
			"override file "+path+" is absent; run `coily lockdown init-config` to install the template"), nil
	}
	if err != nil {
		return Resolution{}, err
	}
	coord, ok := loaded[profileName]
	if !ok {
		path, _ := OverridePath()
		return strictest(SourceUnknownProfile,
			"profile "+profileName+" is not declared in "+path), nil
	}
	return Resolution{
		Coord:  coord,
		Source: SourceOverride,
		Note:   "profile " + profileName + " from override file",
	}, nil
}

func strictest(src Source, note string) Resolution {
	return Resolution{
		Coord:  profile.Strictest(),
		Source: src,
		Note:   note,
	}
}

// validateProfileName mirrors cmd/coily/ops_session.go's check. Lives
// here too so a malformed name in the YAML fails at load time, not
// only when a session sentinel points at it.
func validateProfileName(s string) error {
	if s == "" {
		return errors.New("profile name is empty")
	}
	if len(s) > 40 {
		return fmt.Errorf("profile name %q exceeds 40 chars", s)
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-':
			if i == 0 || i == len(s)-1 {
				return fmt.Errorf("profile name %q must not start or end with '-'", s)
			}
		default:
			return fmt.Errorf("profile name %q contains invalid char %q", s, r)
		}
	}
	return nil
}

// validateRaw enforces the all-axes-required rule and checks each
// declared tier against the cli-guard/profile axis ordering. Returns a
// validated Coordinate or a kind-tagged error.
func validateRaw(name string, raw rawCoordinate) (profile.Coordinate, error) {
	checks := []struct {
		axis  profile.Axis
		got   string
		field string
	}{
		{profile.AxisDataSecurity, raw.DataSecurity, "data_security"},
		{profile.AxisBlastRadius, raw.BlastRadius, "blast_radius"},
		{profile.AxisNetworkEgress, raw.NetworkEgress, "network_egress"},
		{profile.AxisFilesystemReach, raw.FilesystemReach, "filesystem_reach"},
	}
	for _, c := range checks {
		if c.got == "" {
			return profile.Coordinate{}, fmt.Errorf("profile %q is missing required axis %q", name, c.field)
		}
		if !tierAllowed(c.axis, profile.Tier(c.got)) {
			return profile.Coordinate{}, fmt.Errorf("profile %q axis %q: tier %q is not valid (allowed: %v)",
				name, c.field, c.got, profile.TiersFor(c.axis))
		}
	}
	return profile.Coordinate{
		DataSecurity:    profile.Tier(raw.DataSecurity),
		BlastRadius:     profile.Tier(raw.BlastRadius),
		NetworkEgress:   profile.Tier(raw.NetworkEgress),
		FilesystemReach: profile.Tier(raw.FilesystemReach),
	}, nil
}

func tierAllowed(a profile.Axis, t profile.Tier) bool {
	for _, allowed := range profile.TiersFor(a) {
		if allowed == t {
			return true
		}
	}
	return false
}
