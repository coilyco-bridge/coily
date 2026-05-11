// Package workdir does best-effort detection of the "primary working
// directory" that a coily invocation is operating against. Subcommands
// and audit-adjacent code that want a "what repo am I being called from"
// hint should call Detect rather than rederiving it.
//
// This is intentionally separate from pkg/scope. scope is authoritative:
// it produces the commit-scope path that audit rows bind to, and refuses
// to fall back when the answer is uncertain. workdir is a hint: it
// always returns a path, and tags the signal it used so callers can
// downgrade trust when the source is weak.
//
// Signal order (first match wins):
//  1. $COILY_PRIMARY_DIR override - operator forced an answer.
//  2. Nearest ancestor containing a .git entry (file or dir, so worktrees
//     count). Pure filesystem walk, no `git` spawn.
//  3. If cwd is inside ~/projects/coilysiren/, the first path segment
//     under it. Matches Kai's repo-parent workspace shape.
//  4. cwd itself.
package workdir

import (
	"os"
	"path/filepath"
	"strings"
)

// OverrideEnv names the env var that forces Detect's answer.
const OverrideEnv = "COILY_PRIMARY_DIR"

// Source labels which signal produced the result. Callers that need to
// gate behavior on confidence should branch on this.
type Source string

const (
	SourceEnv        Source = "env"
	SourceGit        Source = "git"
	SourceCoilysiren Source = "coilysiren"
	SourceCWD        Source = "cwd"
)

// Result is what Detect returns. Path is always absolute and cleaned.
type Result struct {
	Path   string
	Source Source
}

// Detect runs the signal chain against the current process cwd.
func Detect() Result {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	return DetectFrom(cwd)
}

// DetectFrom is Detect parameterized on cwd, for tests and callers that
// already have a working directory in hand.
func DetectFrom(cwd string) Result {
	if v := strings.TrimSpace(os.Getenv(OverrideEnv)); v != "" {
		return Result{Path: absUnder(cwd, v), Source: SourceEnv}
	}
	if root := findGitRoot(cwd); root != "" {
		return Result{Path: root, Source: SourceGit}
	}
	if repo := coilysirenRepo(cwd); repo != "" {
		return Result{Path: repo, Source: SourceCoilysiren}
	}
	return Result{Path: filepath.Clean(cwd), Source: SourceCWD}
}

func absUnder(cwd, v string) string {
	if !filepath.IsAbs(v) {
		v = filepath.Join(cwd, v)
	}
	return filepath.Clean(v)
}

// findGitRoot walks from start toward the filesystem root, returning the
// first directory that has a .git child (file or dir). Worktree
// checkouts use a .git file pointing at the real gitdir, so a stat-only
// check is enough.
func findGitRoot(start string) string {
	if start == "" {
		return ""
	}
	dir := filepath.Clean(start)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// coilysirenRepo returns ~/projects/coilysiren/<first-segment> when cwd
// is inside that parent, else "".
func coilysirenRepo(cwd string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	parent := filepath.Join(home, "projects", "coilysiren")
	rel, err := filepath.Rel(parent, filepath.Clean(cwd))
	if err != nil {
		return ""
	}
	if rel == "." || rel == "" || strings.HasPrefix(rel, "..") {
		return ""
	}
	segs := strings.Split(rel, string(filepath.Separator))
	if len(segs) == 0 || segs[0] == "" {
		return ""
	}
	return filepath.Join(parent, segs[0])
}
