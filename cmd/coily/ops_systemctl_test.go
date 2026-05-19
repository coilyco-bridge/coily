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
// The `scope` field is the outer's already-resolved --commit-scope flag
// value, forwarded verbatim to the inner. #244 made this an explicit
// passthrough rather than re-resolving from cwd, because the cli-guard
// scope resolver only bypasses the git check for non-"auto" values.
func TestBuildSelfElevateArgv(t *testing.T) {
	cases := []struct {
		name      string
		unit      string
		scope     string
		needsUnit bool
		want      []string
	}{
		{
			name: "stop", unit: "sirens-discord-ops-update.timer",
			scope: "/home/kai/projects/coilysiren/infrastructure", needsUnit: true,
			want: []string{"sudo", "--non-interactive", "/home/linuxbrew/.linuxbrew/bin/coily",
				"--commit-scope=/home/kai/projects/coilysiren/infrastructure",
				"systemctl", "stop", "sirens-discord-ops-update.timer"},
		},
		{
			name:  "daemon-reload",
			scope: "/home/kai/projects/coilysiren/infrastructure", needsUnit: false,
			want: []string{"sudo", "--non-interactive", "/home/linuxbrew/.linuxbrew/bin/coily",
				"--commit-scope=/home/kai/projects/coilysiren/infrastructure",
				"systemctl", "daemon-reload"},
		},
		{
			// coily#244: caller in the "auto" / unset case maps to
			// empty scope at the call site (see systemctlSelfElevate),
			// so the inner falls back to its own auto resolution.
			name: "empty-scope", unit: "x.service", scope: "", needsUnit: true,
			want: []string{"sudo", "--non-interactive", "/home/linuxbrew/.linuxbrew/bin/coily",
				"systemctl", "empty-scope", "x.service"},
		},
		{
			// coily#244: coily ssh injects --commit-scope=<working_dir>
			// (a non-git path like /home/kai/projects/coilysiren) on
			// the outer. That value gets forwarded verbatim so the
			// inner takes the same explicit-path bypass branch.
			name: "ssh-working-dir", unit: "repo-recall.service",
			scope: "/home/kai/projects/coilysiren", needsUnit: true,
			want: []string{"sudo", "--non-interactive", "/home/linuxbrew/.linuxbrew/bin/coily",
				"--commit-scope=/home/kai/projects/coilysiren",
				"systemctl", "ssh-working-dir", "repo-recall.service"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildSelfElevateArgv("/home/linuxbrew/.linuxbrew/bin/coily", tc.scope, tc.name, tc.unit, tc.needsUnit)
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

// TestBuildSelfElevateArgv_OmitsCommitScopeWhenEmpty pins the coily#244
// regression fix: when the caller passes empty scope (because the outer
// had --commit-scope=auto, which the caller maps to ""), the rendered
// argv must NOT contain `--commit-scope=...`. The inner then falls back
// to its own default ("auto"), which is the right behavior for any
// caller that genuinely has no scope to forward.
func TestBuildSelfElevateArgv_OmitsCommitScopeWhenEmpty(t *testing.T) {
	argv := buildSelfElevateArgv("/home/linuxbrew/.linuxbrew/bin/coily", "", "disable", "repo-recall.service", true)
	for _, a := range argv {
		if strings.HasPrefix(a, "--commit-scope") {
			t.Errorf("found %q in argv with empty scope; coily#244 forbids --commit-scope when scope is empty (full=%v)", a, argv)
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
