package sudo

import (
	"errors"
	"os"
	"runtime"
	"testing"
)

// TestReadPassword_NoTTY verifies that ReadPassword fails fast with
// ErrNoTTY when /dev/tty is unreachable, rather than hanging. The
// no-tty case is the hot path for CI and any non-interactive invocation
// of `coily ssh deploy`; hanging would defeat the -n-first design.
func TestReadPassword_NoTTY(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no /dev/tty on windows")
	}
	// chroot-style sandboxes don't exist in stdlib tests, so we approximate
	// "no tty" by closing stdin's controlling tty. The robust check is just
	// to ensure that when /dev/tty open fails, we get ErrNoTTY rather than
	// a hang. We can't reliably make the open fail on a developer machine,
	// so verify the error path by stat-ing /dev/tty: if it doesn't exist
	// (sandboxed CI), ReadPassword must return ErrNoTTY.
	if _, err := os.Stat("/dev/tty"); err == nil {
		t.Skip("dev has /dev/tty; ErrNoTTY path only fires under sandboxed CI")
	}
	_, err := ReadPassword("test: ")
	if !errors.Is(err, ErrNoTTY) {
		t.Fatalf("ReadPassword without tty: err=%v, want ErrNoTTY", err)
	}
}

func TestZero(t *testing.T) {
	b := []byte("hunter2")
	Zero(b)
	for i, v := range b {
		if v != 0 {
			t.Fatalf("Zero left byte %d = %d", i, v)
		}
	}
}

func TestPasswordRequired(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"sudo: a password is required\n", true},
		{"sudo: a terminal is required to read the password\n", true},
		{"Sorry, a password is required to run sudo\n", true}, // matches via the "a password is required" substring
		{"sorry, a password is required to run sudo\n", true},
		{"bash: /home/kai/.../install.sh: No such file or directory\n", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := PasswordRequired(tc.in); got != tc.want {
				t.Errorf("PasswordRequired(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
