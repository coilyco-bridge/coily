package main

import "testing"

// TestResolveInvokeCWD_PrefersExplicitOverOldpwdOverGetwd pins the
// resolution order from coilysiren/coily#109 so an agent can stop a
// verb's audit row from binding to the post-cd subprocess cwd by
// exporting COILY_INVOKE_CWD or relying on the bash-cd $OLDPWD.
func TestResolveInvokeCWD_PrefersExplicitOverOldpwdOverGetwd(t *testing.T) {
	parent := t.TempDir()
	other := t.TempDir()

	t.Setenv("COILY_INVOKE_CWD", parent)
	t.Setenv("OLDPWD", other)
	if got := resolveInvokeCWD(); got != parent {
		t.Errorf("with COILY_INVOKE_CWD set, got %q, want %q", got, parent)
	}

	t.Setenv("COILY_INVOKE_CWD", "")
	t.Setenv("OLDPWD", other)
	if got := resolveInvokeCWD(); got != other {
		t.Errorf("with only OLDPWD set, got %q, want %q", got, other)
	}

	t.Setenv("COILY_INVOKE_CWD", "/nonexistent/path/should/skip")
	t.Setenv("OLDPWD", other)
	if got := resolveInvokeCWD(); got != other {
		t.Errorf("stale COILY_INVOKE_CWD should be skipped; got %q, want %q", got, other)
	}
}
