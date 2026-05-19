package main

import (
	"sort"
	"strings"
	"testing"
)

// TestBuildSelfElevateArgv pins the coily#203 invariant: when `coily
// systemctl <verb> <unit>` self-elevates, the rendered argv has outer
// sudo on the coily binary and never an inner `sudo systemctl ...`.
// --non-interactive is required so a missing NOPASSWD grant fails fast
// instead of waiting on a hidden password prompt.
//
// The `toplevel` field represents the outer process's git toplevel
// after gitToplevel() resolution (empty when the outer cwd is not
// inside any git repo). #244 added the empty-toplevel branch: omitting
// --commit-scope so the inner takes the auto -> _unrooted fallthrough.
func TestBuildSelfElevateArgv(t *testing.T) {
	cases := []struct {
		name      string
		unit      string
		toplevel  string
		needsUnit bool
		want      []string
	}{
		{
			name: "stop", unit: "sirens-discord-ops-update.timer",
			toplevel: "/home/kai/projects/coilysiren/infrastructure", needsUnit: true,
			want: []string{"sudo", "--non-interactive", "/home/linuxbrew/.linuxbrew/bin/coily",
				"--commit-scope=/home/kai/projects/coilysiren/infrastructure",
				"systemctl", "stop", "sirens-discord-ops-update.timer"},
		},
		{
			name:     "daemon-reload",
			toplevel: "/home/kai/projects/coilysiren/infrastructure", needsUnit: false,
			want: []string{"sudo", "--non-interactive", "/home/linuxbrew/.linuxbrew/bin/coily",
				"--commit-scope=/home/kai/projects/coilysiren/infrastructure",
				"systemctl", "daemon-reload"},
		},
		{
			// coily#244: outer cwd outside a git repo (e.g. /home/kai
			// via `coily ssh kai-server -- coily systemctl ...`) means
			// gitToplevel returns "". No --commit-scope is passed so
			// the inner falls through to auto/_unrooted same as the outer.
			name: "no-toplevel", unit: "x.service", toplevel: "", needsUnit: true,
			want: []string{"sudo", "--non-interactive", "/home/linuxbrew/.linuxbrew/bin/coily",
				"systemctl", "no-toplevel", "x.service"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildSelfElevateArgv("/home/linuxbrew/.linuxbrew/bin/coily", tc.toplevel, tc.name, tc.unit, tc.needsUnit)
			if len(got) != len(tc.want) {
				t.Fatalf("argv length = %d, want %d (got=%v)", len(got), len(tc.want), got)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("argv[%d] = %q, want %q (full=%v)", i, got[i], tc.want[i], got)
				}
			}
			for i, a := range got {
				if i > 0 && a == "sudo" {
					t.Errorf("inner sudo at argv[%d]; outer sudo only (coily#203)", i)
				}
			}
		})
	}
}

// TestBuildSelfElevateArgv_OmitsCommitScopeWhenNotInGitRepo pins the
// coily#244 regression fix: when the caller's git-toplevel resolution
// fails (cwd is not inside any git repo), the rendered argv must NOT
// contain `--commit-scope=...`. Passing a non-git path explicitly trips
// the inner cli-guard strict-mode scope gate, which surfaces as a
// confusing "scope: cwd is not inside a git repo" error inside the
// sudo'd child. Omitting the flag entirely lets the inner take the
// same auto -> _unrooted fallthrough the outer just took.
func TestBuildSelfElevateArgv_OmitsCommitScopeWhenNotInGitRepo(t *testing.T) {
	argv := buildSelfElevateArgv("/home/linuxbrew/.linuxbrew/bin/coily", "", "disable", "repo-recall.service", true)
	for _, a := range argv {
		if strings.HasPrefix(a, "--commit-scope") {
			t.Errorf("found %q in argv with empty toplevel; coily#244 forbids explicit scope when cwd has no git toplevel (full=%v)", a, argv)
		}
	}
}

// TestSecurityClaim_SystemctlSelfElevatesNotInnerSudo backs the coily#203
// design intent: coily is the security boundary, the broad
// `(ALL) NOPASSWD: <coily-path>` sudoers rule is the trusted grant, and
// per-unit sudoers carveouts duplicate the gate. The self-elevation
// shape must therefore put sudo on coily, not on systemctl. Drift in
// either direction (inner sudo, or unprefixed sudo on systemctl) trips
// this test.
func TestSecurityClaim_SystemctlSelfElevatesNotInnerSudo(t *testing.T) {
	argv := buildSelfElevateArgv("/some/path/coily", "/some/repo", "stop", "x.service", true)
	if argv[0] != "sudo" {
		t.Errorf("argv[0] = %q, want \"sudo\" (outer sudo on coily)", argv[0])
	}
	if !strings.HasSuffix(argv[2], "coily") {
		t.Errorf("argv[2] = %q, want a coily binary path", argv[2])
	}
	var sawSystemctl bool
	for i, a := range argv {
		if i > 0 && a == "sudo" {
			t.Errorf("found inner sudo at argv[%d]; coily#203 forbids per-systemctl sudo carveouts", i)
		}
		if a == "systemctl" {
			sawSystemctl = true
		}
	}
	if !sawSystemctl {
		t.Errorf("argv missing systemctl token; got %v", argv)
	}
}

// TestSystemctlCommand_HasAllVerbs pins the closed-set surface so a future
// addition to systemctlVerbs auto-shows up in `coily systemctl` (and a
// future removal can't accidentally silently land).
func TestSystemctlCommand_HasAllVerbs(t *testing.T) {
	r := &Runner{}
	cmd := r.systemctlCommand()
	if cmd.Name != "systemctl" {
		t.Fatalf("Name = %q", cmd.Name)
	}
	got := make([]string, 0, len(cmd.Commands))
	for _, c := range cmd.Commands {
		got = append(got, c.Name)
	}
	sort.Strings(got)
	want := []string{"daemon-reload", "disable", "enable", "restart", "start", "status", "stop"}
	if len(got) != len(want) {
		t.Fatalf("verbs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("verbs[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
