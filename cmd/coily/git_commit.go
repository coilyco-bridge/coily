// git_commit.go is the concurrency-safe replacement for the thin
// `coily git commit` passthrough (coily#7).
//
// The bug it fixes: two interactive agent sessions share one working tree
// (the bare ~/projects/coilysiren/<repo> checkout). The everyday workflow
// is two separate coily invocations - `coily git add X` then `coily git
// commit -m msg` - and a second session's add/commit interleaves in the
// gap. Because the staging area (`.git/index`) and `.git/COMMIT_EDITMSG`
// are shared process-global files, content from one session lands under the
// other session's message. Commit 4dd385c crossed exactly this way.
//
// The fix removes the shared index and the editor from the commit path:
//
//   - The commit must name its pathspecs (`coily git commit -m msg -- a b`).
//     Git's pathspec-commit mode commits the *worktree* content of exactly
//     those paths, seeded from HEAD, disregarding whatever else is staged in
//     the shared index. Another session's staged files cannot leak in.
//   - We point GIT_INDEX_FILE at a private throwaway index, so the commit
//     never reads or writes the shared `.git/index` at all - no contention,
//     no drift caused by us. We seed that private index from HEAD and stage
//     just the named paths into it, so a brand-new file (no HEAD entry) can
//     be committed too - the empty-index case used to refuse new files
//     (coily#192).
//   - The message must come from `-m`/`-F` (never the editor), so
//     `.git/COMMIT_EDITMSG` is written-not-read and each commit's message is
//     taken from its own argv. Messages cannot cross.
//
// After the commit we resync only the named paths in the shared index to
// the new HEAD (`git reset -q HEAD -- <paths>`), best-effort, so the next
// `git status` in the shared tree reads clean for what we just committed
// without disturbing another session's staged entries.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/shell"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// gitCommitCommand builds the dedicated `coily git commit`. SkipFlagParsing
// is on so git's own (large, evolving) flag surface passes through verbatim;
// coily only inspects the argv enough to enforce the two safety invariants
// (explicit pathspecs, non-editor message) and to hoist a leading -C.
func (r *Runner) gitCommitCommand() *cli.Command {
	return &cli.Command{
		Name:            "commit",
		Usage:           "git commit - record named paths atomically (concurrency-safe, coily#7).",
		SkipFlagParsing: true,
		Description: `commit records changes for explicitly named paths in a way that is
safe when multiple sessions share one working tree.

Unlike a raw 'git commit', it requires the form:

  coily git commit -m "msg" -- <path> [<path>...]

The paths after '--' are committed from the working tree, seeded from
HEAD, against a private index - so another session's staged files never
leak into your commit, and your message (which must come from -m/-F,
never the editor) never crosses with theirs. A leading '-C <dir>'
selects a repo other than the current directory.`,
		Action: r.WrapVerb(
			verb.Spec{
				Name:       "git.commit",
				SkipPolicy: true,
				Action: func(ctx context.Context, c *cli.Command) error {
					return r.runGitCommit(ctx, c.Args().Slice())
				},
			},
			r.Audit,
		),
	}
}

// runGitCommit validates argv, then stages+commits the named paths against
// a private index and resyncs the shared index for those paths.
func (r *Runner) runGitCommit(ctx context.Context, argv []string) error {
	dir, rest := hoistDashC(argv)

	flags, paths, err := splitCommitArgs(rest)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("coily git commit: name the files to commit after '--', e.g. " +
			"`coily git commit -m \"msg\" -- path/to/file` - bare commit is refused so a " +
			"concurrent session's staged files cannot cross into this commit (coily#7)")
	}
	if !hasMessageSource(flags) {
		return fmt.Errorf("coily git commit: pass -m/-F (the editor is refused under a shared " +
			"working tree so messages cannot cross between sessions - coily#7)")
	}
	if hasEditorFlag(flags) {
		return fmt.Errorf("coily git commit: -e/--edit is refused under a shared working tree; " +
			"supply the message with -m/-F (coily#7)")
	}

	idxPath, cleanup, err := newPrivateIndex()
	if err != nil {
		return err
	}
	defer cleanup()

	// Private-index runner: identical streams to the shared runner, plus
	// GIT_INDEX_FILE so this commit never touches the shared .git/index.
	idxRunner := &shell.Runner{
		Stdout:  r.Runner.Stdout,
		Stderr:  r.Runner.Stderr,
		Stdin:   r.Runner.Stdin,
		Resolve: r.Runner.Resolve,
		Env:     []string{"GIT_INDEX_FILE=" + idxPath},
	}

	// Seed the private index from HEAD, then stage the worktree state of
	// exactly the named paths into it. The seed matters for NEW files: git's
	// pathspec-commit (`commit -- <paths>`) re-stages each path with
	// `update-index <path>`, which fails with "did not match any file(s)
	// known to git" when the path is neither in HEAD nor in the index - the
	// empty-index case (coily#192). Seeding from HEAD plus an explicit
	// `git add` gives every named path an index entry to update, so new,
	// modified, and deleted paths all commit uniformly. Other HEAD entries
	// are carried through untouched, and no concurrently-staged path from the
	// shared index can leak in, because this index only ever sees HEAD + the
	// named paths.
	if err := seedPrivateIndex(ctx, idxRunner, dir, paths); err != nil {
		return err
	}

	commitArgv := gitArgv(dir, "commit", flags, paths)
	if err := idxRunner.Exec(ctx, "git", commitArgv...); err != nil {
		return fmt.Errorf("coily git commit: %w", err)
	}

	// Best-effort: resync only the committed paths in the shared index to
	// the new HEAD so the next `git status` reads clean for them. A failure
	// here (e.g. a transient index.lock from another session, or no HEAD on
	// the very first commit) does not undo the commit, so warn and move on.
	resyncArgv := gitArgv(dir, "reset", []string{"-q", "HEAD"}, paths)
	if err := r.Runner.Exec(ctx, "git", resyncArgv...); err != nil {
		fmt.Fprintf(os.Stderr, "coily git commit: committed, but resyncing the shared index for "+
			"the committed paths failed (%v); `git status` may show them until the next index "+
			"update\n", err)
	}
	return nil
}

// seedPrivateIndex primes the private index so a pathspec-commit of the named
// paths succeeds for new files as well as modifications and deletions. It
// reads HEAD into the index (best-effort: the very first commit has no HEAD,
// where an empty index is already correct), then runs `git add -A -- <paths>`
// to stage the worktree state of exactly those paths. Only the named paths are
// staged, so nothing else from the working tree - or another session's shared
// index - can ride along.
func seedPrivateIndex(ctx context.Context, idxRunner *shell.Runner, dir string, paths []string) error {
	// read-tree HEAD through a stderr-discarding copy of the runner: the very
	// first commit has no HEAD, where read-tree fails and an empty index is
	// already correct, so we swallow both the error and git's fatal line.
	readTreeArgv := []string{}
	if dir != "" {
		readTreeArgv = append(readTreeArgv, "-C", dir)
	}
	readTreeArgv = append(readTreeArgv, "read-tree", "HEAD")
	quiet := *idxRunner
	quiet.Stderr = io.Discard
	_ = quiet.Exec(ctx, "git", readTreeArgv...)

	addArgv := gitArgv(dir, "add", []string{"-A"}, paths)
	if err := idxRunner.Exec(ctx, "git", addArgv...); err != nil {
		return fmt.Errorf("coily git commit: staging the named paths failed: %w", err)
	}
	return nil
}

// hoistDashC peels a leading `-C <dir>` off argv (the coily convention for
// operating on a repo other than cwd) and returns it plus the remainder.
// git wants -C before the subcommand, so coily owns its placement.
func hoistDashC(argv []string) (dir string, rest []string) {
	if len(argv) >= 2 && argv[0] == "-C" {
		return argv[1], argv[2:]
	}
	return "", argv
}

// splitCommitArgs partitions the post-(-C) argv at the first `--` into the
// flags git commit receives and the pathspecs to commit. A `--` is required:
// the explicit separator is what makes "commit exactly these paths"
// unambiguous regardless of how git's flag surface evolves.
func splitCommitArgs(argv []string) (flags, paths []string, err error) {
	for i, a := range argv {
		if a == "--" {
			return argv[:i], argv[i+1:], nil
		}
	}
	return nil, nil, fmt.Errorf("coily git commit: missing '--' separator; use " +
		"`coily git commit -m \"msg\" -- <path>...` so the files to commit are explicit (coily#7)")
}

// hasMessageSource reports whether the flags carry a non-interactive commit
// message, i.e. some -m/--message or -F/--file form. Without one, git would
// drop to the editor, which is exactly the shared-COMMIT_EDITMSG hazard.
func hasMessageSource(flags []string) bool {
	for _, f := range flags {
		switch {
		case f == "-m" || f == "-F":
			return true
		case strings.HasPrefix(f, "-m") && len(f) > 2: // -m"msg"
			return true
		case strings.HasPrefix(f, "-F") && len(f) > 2: // -Ffile
			return true
		case f == "--message" || strings.HasPrefix(f, "--message="):
			return true
		case f == "--file" || strings.HasPrefix(f, "--file="):
			return true
		}
	}
	return false
}

// hasEditorFlag reports whether the flags explicitly request the editor,
// which is refused even when a message source is also present.
func hasEditorFlag(flags []string) bool {
	for _, f := range flags {
		if f == "-e" || f == "--edit" {
			return true
		}
	}
	return false
}

// gitArgv assembles `[-C <dir>] <verb> <flags...> -- <paths...>`. The -C, when
// set, leads so it lands before the subcommand as git requires.
func gitArgv(dir, verbName string, flags, paths []string) []string {
	out := make([]string, 0, len(flags)+len(paths)+4)
	if dir != "" {
		out = append(out, "-C", dir)
	}
	out = append(out, verbName)
	out = append(out, flags...)
	out = append(out, "--")
	out = append(out, paths...)
	return out
}

// newPrivateIndex reserves a unique path for a throwaway GIT_INDEX_FILE and
// returns a cleanup that removes it. The placeholder file is deleted before
// return so git creates the index fresh; seedPrivateIndex then primes it from
// HEAD and stages the named paths (see coily#192 for why an absent index alone
// was not enough - it refused new files).
func newPrivateIndex() (path string, cleanup func(), err error) {
	f, err := os.CreateTemp("", "coily-gitidx-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("coily git commit: create private index: %w", err)
	}
	path = f.Name()
	_ = f.Close()
	_ = os.Remove(path)
	return path, func() { _ = os.Remove(path) }, nil
}
