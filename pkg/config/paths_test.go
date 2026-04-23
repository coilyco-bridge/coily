package config_test

import (
	"os"
	"testing"

	"github.com/coilysiren/coily/pkg/config"
)

// chdirT swaps the process cwd to dir for the test's lifetime. Go 1.24
// added testing.Chdir; we support 1.23 so do it the manual way.
func chdirT(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func TestSanitizeSlug(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"coilysiren-backend", "coilysiren-backend"},
		{"CoilySiren-Backend", "coilysiren-backend"},
		{"coilysiren/backend", "coilysiren-backend"},
		{"My Org/Some Repo", "my-org-some-repo"},
		{"foo/bar.baz", "foo-bar-baz"},
		{"under_score/repo", "under-score-repo"},
		{"trailing/dashes--", "trailing-dashes"},
		{"___", "_unrooted"},
		{"", "_unrooted"},
		{"weird!!chars##here", "weird-chars-here"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := config.SanitizeSlug(tc.in)
			if got != tc.want {
				t.Errorf("SanitizeSlug(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestRepoAuditSlug_FallbackOutsideGit(t *testing.T) {
	// Run from a temp directory that is not a git repo. Slug should fall back
	// to the unrooted sentinel.
	dir := t.TempDir()
	chdirT(t, dir)
	config.ResetRepoSlugCacheForTest()
	got := config.RepoAuditSlug()
	if got != config.UnrootedAuditName {
		t.Errorf("RepoAuditSlug() = %q outside a git repo, want %q", got, config.UnrootedAuditName)
	}
	config.ResetRepoSlugCacheForTest()
}
