// Package decision is the per-call profile-aware evaluator coily
// injects into verb.Spec.OnEvaluate when audit.profile_aware is true.
// Phase 4 of coilysiren/coily#150 ships pure plumbing: every call
// returns Allowed=true with the resolved Coordinate attached so the
// audit log gathers a soak signal before phase 5 picks the first axis
// to enforce.
package decision

import (
	"github.com/coilysiren/cli-guard/audit"
	"github.com/coilysiren/cli-guard/profile"
	"github.com/coilysiren/coily/pkg/profiles"
)

// Evaluate resolves the named session profile via pkg/profiles and
// returns an attached audit.ProfileDecision. Allowed is always true in
// phase 4 (pure plumbing). The error return path is reserved for
// loader failures (malformed ~/.coily/coily.yaml); a missing override
// or an unknown profile name fall through to a Strictest Coordinate
// with the matching Source value rather than an error.
func Evaluate(profileName string) (*audit.ProfileDecision, error) {
	res, err := profiles.Resolve(profileName)
	if err != nil {
		return nil, err
	}
	return &audit.ProfileDecision{
		Allowed:    true,
		Profile:    profileName,
		Source:     string(res.Source),
		Coordinate: snapshot(res.Coord),
		Reason:     res.Note,
	}, nil
}

// CoordinatePtr returns a non-nil *profile.Coordinate from the
// resolver result, suitable for lockdown.Driver.Coordinate. Callers
// that want the Strictest-fallback behavior on missing override can
// pass the empty profile name; the resolver hands them Strictest.
func CoordinatePtr(profileName string) (*profile.Coordinate, error) {
	res, err := profiles.Resolve(profileName)
	if err != nil {
		return nil, err
	}
	c := res.Coord
	return &c, nil
}

func snapshot(c profile.Coordinate) audit.Coordinate {
	return audit.Coordinate{
		DataSecurity:    string(c.DataSecurity),
		BlastRadius:     string(c.BlastRadius),
		NetworkEgress:   string(c.NetworkEgress),
		FilesystemReach: string(c.FilesystemReach),
	}
}
