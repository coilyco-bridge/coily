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
func TestBuildSelfElevateArgv(t *testing.T) {
	cases := []struct {
		name      string
		unit      string
		cwd       string
		needsUnit bool
		want      []string
	}{
		{
			name: "stop", unit: "sirens-discord-ops-update.timer",
			cwd: "/home/kai/projects/coilysiren/infrastructure", needsUnit: true,
			want: []string{"sudo", "--non-interactive", "/home/linuxbrew/.linuxbrew/bin/coily",
				"--commit-scope=/home/kai/projects/coilysiren/infrastructure",
				"systemctl", "stop", "sirens-discord-ops-update.timer"},
		},
		{
			name: "daemon-reload",
			cwd:  "/home/kai/projects/coilysiren/infrastructure", needsUnit: false,
			want: []string{"sudo", "--non-interactive", "/home/linuxbrew/.linuxbrew/bin/coily",
				"--commit-scope=/home/kai/projects/coilysiren/infrastructure",
				"systemctl", "daemon-reload"},
		},
		{
			name: "no-cwd", unit: "x.service", cwd: "", needsUnit: true,
			want: []string{"sudo", "--non-interactive", "/home/linuxbrew/.linuxbrew/bin/coily",
				"systemctl", "no-cwd", "x.service"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildSelfElevateArgv("/home/linuxbrew/.linuxbrew/bin/coily", tc.cwd, tc.name, tc.unit, tc.needsUnit)
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
