package main

import "testing"

func TestGHReadCacheClassifier(t *testing.T) {
	cases := []struct {
		name     string
		argv     []string
		wantPath string
		wantOK   bool
	}{
		// Reads - post-rewriter shapes.
		{
			name:     "bare api path",
			argv:     []string{"api", "/repos/o/r/issues/1"},
			wantPath: "/repos/o/r/issues/1",
			wantOK:   true,
		},
		{
			name:     "api path without leading slash",
			argv:     []string{"api", "repos/o/r/issues/1"},
			wantPath: "repos/o/r/issues/1",
			wantOK:   true,
		},
		{
			name:     "explicit -X GET",
			argv:     []string{"api", "-X", "GET", "/repos/o/r/issues"},
			wantPath: "/repos/o/r/issues",
			wantOK:   true,
		},
		{
			name:     "explicit --method=get lowercase",
			argv:     []string{"api", "--method=get", "/repos/o/r"},
			wantPath: "/repos/o/r",
			wantOK:   true,
		},

		// Writes - decline.
		{
			name:   "POST via -X",
			argv:   []string{"api", "-X", "POST", "repos/o/r/issues/1/comments", "-f", "body=hi"},
			wantOK: false,
		},
		{
			name:   "PATCH via --method",
			argv:   []string{"api", "--method", "PATCH", "repos/o/r/issues/1", "-f", "state=closed"},
			wantOK: false,
		},
		{
			name:   "DELETE via -X=",
			argv:   []string{"api", "-X=DELETE", "/repos/o/r/issues/1"},
			wantOK: false,
		},
		{
			name:   "-f body flag without explicit method",
			argv:   []string{"api", "/repos/o/r/issues", "-f", "title=foo"},
			wantOK: false,
		},
		{
			name:   "-F body flag",
			argv:   []string{"api", "/repos/o/r/issues", "-F", "title=@-"},
			wantOK: false,
		},
		{
			name:   "--field body flag",
			argv:   []string{"api", "/repos/o/r/issues", "--field", "title=foo"},
			wantOK: false,
		},
		{
			name:   "--input body source",
			argv:   []string{"api", "/repos/o/r/issues", "--input", "-"},
			wantOK: false,
		},
		{
			name:   "--field= form",
			argv:   []string{"api", "/repos/o/r/issues", "--field=title=foo"},
			wantOK: false,
		},

		// Edge cases - decline.
		{
			name:   "first token not api",
			argv:   []string{"issue", "list", "--repo", "o/r"},
			wantOK: false,
		},
		{
			name:   "just api with no path",
			argv:   []string{"api"},
			wantOK: false,
		},
		{
			name:   "unknown flag",
			argv:   []string{"api", "--paginate", "/repos/o/r/issues"},
			wantOK: false,
		},
		{
			name:   "two positionals",
			argv:   []string{"api", "/a", "/b"},
			wantOK: false,
		},
		{
			name:   "empty argv",
			argv:   nil,
			wantOK: false,
		},
		{
			name:   "-X with no value",
			argv:   []string{"api", "-X"},
			wantOK: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotOK := ghReadCacheClassifier(tc.argv)
			if gotOK != tc.wantOK {
				t.Errorf("ok=%v want %v (path=%q)", gotOK, tc.wantOK, gotPath)
			}
			if tc.wantOK && gotPath != tc.wantPath {
				t.Errorf("path=%q want %q", gotPath, tc.wantPath)
			}
		})
	}
}
