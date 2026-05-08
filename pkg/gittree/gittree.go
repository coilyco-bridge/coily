// Package gittree inspects a repo's working tree for the clean+synced state
// that gates `.coily/coily.yaml` repo verbs. The gate refuses repo-verb
// invocations when the audit log could not be reconstructed from git history
// alone: uncommitted changes, untracked files, detached HEAD, or a branch
// that has not been fetched against its upstream recently.
//
// The gate is repo-verb-only. Built-in coily verbs are reproducible from the
// binary version trailer in the audit row and are not subject to this check.
package gittree

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// State is the outcome of CheckClean. Clean is true when every gate property
// holds. When false, Reason names the first failure in human-readable form
// and Recovery names the dictatable shell command(s) the operator should run.
// Status is the raw porcelain output (truncated) so a forensic reader can
// reconstruct what was outstanding at refusal time.
type State struct {
	Clean    bool
	Reason   string
	Recovery string
	Status   string
	Branch   string
	Upstream string
	Ahead    int
	Behind   int
}

// MaxStatusBytes caps Status so a sprawling untracked tree does not bloat
// the audit row or refusal message.
const MaxStatusBytes = 2048

// ErrNotGitRepo is returned when the supplied path is not inside a git repo.
// Repo verbs only fire when a coily.yaml was discovered, which itself implies
// a repo, so this error is exceptional rather than a normal gate refusal.
var ErrNotGitRepo = errors.New("gittree: path is not inside a git repo")

// CheckClean evaluates the gate at repoRoot. Returns a *State whose Clean
// field tells the caller whether the repo verb may run. A non-nil error is
// returned only for environmental failures (git missing, repoRoot is not a
// git repo) - normal "tree is dirty" outcomes are reported via State, not
// error, so the caller can format a tailored refusal message.
func CheckClean(repoRoot string) (*State, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("gittree: git binary not found on $PATH: %w", err)
	}
	if out, gerr := runGit(repoRoot, "rev-parse", "--is-inside-work-tree"); gerr != nil || strings.TrimSpace(out) != "true" {
		return nil, fmt.Errorf("%w: %s", ErrNotGitRepo, repoRoot)
	}

	status, err := runGit(repoRoot, "status", "--porcelain=v1", "--untracked-files=normal")
	if err != nil {
		return nil, fmt.Errorf("gittree: git status: %w", err)
	}
	branch, err := runGit(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("gittree: git rev-parse HEAD: %w", err)
	}

	st := &State{Status: truncate(status, MaxStatusBytes), Branch: strings.TrimSpace(branch)}
	if checkLocalState(st, status) {
		return st, nil
	}
	if checkUpstreamState(st, repoRoot) {
		return st, nil
	}
	return st, checkAheadBehind(st, repoRoot)
}

// checkLocalState fills st with a refusal reason from local-only signals
// (dirty tree, detached HEAD). Returns true when a refusal was set.
func checkLocalState(st *State, status string) bool {
	if status != "" {
		st.Reason = "working tree is dirty"
		st.Recovery = recoveryDirty(st.Status)
		return true
	}
	if st.Branch == "HEAD" {
		st.Reason = "HEAD is detached (no branch)"
		st.Recovery = "  git checkout <branch>\n"
		return true
	}
	return false
}

// checkUpstreamState resolves the branch's upstream and runs `git fetch`.
// Sets a refusal reason on st when the branch has no upstream or fetch
// fails. Returns true when a refusal was set.
func checkUpstreamState(st *State, repoRoot string) bool {
	upstream, err := runGit(repoRoot, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		st.Reason = fmt.Sprintf("branch %q has no upstream", st.Branch)
		st.Recovery = fmt.Sprintf("  git push -u origin %s\n", st.Branch)
		return true
	}
	st.Upstream = strings.TrimSpace(upstream)
	if _, ferr := runGit(repoRoot, "fetch", "--quiet", "--", remoteOf(st.Upstream)); ferr != nil {
		st.Reason = fmt.Sprintf("git fetch failed for %s", remoteOf(st.Upstream))
		st.Recovery = fmt.Sprintf("  git fetch %s\n", remoteOf(st.Upstream))
		return true
	}
	return false
}

// checkAheadBehind populates st.Ahead/st.Behind from `git rev-list`. Sets
// st.Clean=true when the branch is not behind upstream. Returns a non-nil
// error only for environmental failures.
func checkAheadBehind(st *State, repoRoot string) error {
	revs, err := runGit(repoRoot, "rev-list", "--left-right", "--count", "HEAD..."+st.Upstream)
	if err != nil {
		return fmt.Errorf("gittree: git rev-list: %w", err)
	}
	ahead, behind, err := parseAheadBehind(revs)
	if err != nil {
		return err
	}
	st.Ahead = ahead
	st.Behind = behind
	if behind > 0 {
		st.Reason = fmt.Sprintf("%d commits behind %s", behind, st.Upstream)
		st.Recovery = "  git pull --ff-only\n"
		return nil
	}
	st.Clean = true
	return nil
}

// FormatRefusal renders the human-readable refusal message for a non-clean
// State, naming the verb so the operator's recovery suggestion includes
// the retry command. Use this for the stderr line; the caller is responsible
// for wrapping into an exitcode.New so the process exit reflects the gate.
func (s *State) FormatRefusal(verbName string) string {
	if s.Clean {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "refusing repo verb %q - %s\n", verbName, s.Reason)
	if s.Status != "" && strings.HasPrefix(s.Reason, "working tree") {
		b.WriteString(s.Status)
		if !strings.HasSuffix(s.Status, "\n") {
			b.WriteString("\n")
		}
	}
	b.WriteString("\nRepo verbs require a clean tree so the audit log can be reconstructed\n")
	b.WriteString("from git history. Recover with:\n\n")
	b.WriteString(s.Recovery)
	fmt.Fprintf(&b, "  coily %s   # retry\n", verbName)
	b.WriteString("\nOverride for genuine emergencies with --audit-override-dirty.\n")
	b.WriteString("The audit row is tagged audit_override=true and captures the working\n")
	b.WriteString("tree status so the run can still be reconstructed after the fact.")
	return b.String()
}

func recoveryDirty(status string) string {
	hasTracked := false
	hasUntracked := false
	for _, line := range strings.Split(status, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "??") {
			hasUntracked = true
		} else {
			hasTracked = true
		}
	}
	var b strings.Builder
	b.WriteString("  git status              # see what's outstanding\n")
	if hasTracked {
		b.WriteString("  git add ... && git commit\n")
	}
	if hasUntracked {
		b.WriteString("  git add <untracked> && git commit   # or add to .gitignore\n")
	}
	b.WriteString("  git push\n")
	return b.String()
}

func runGit(repoRoot string, args ...string) (string, error) {
	full := append([]string{"-C", repoRoot}, args...)
	cmd := exec.Command("git", full...) // #nosec G204 -- args are coily-controlled, not user-shaped
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func parseAheadBehind(revListOutput string) (ahead, behind int, err error) {
	fields := strings.Fields(strings.TrimSpace(revListOutput))
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("gittree: unexpected rev-list output %q", revListOutput)
	}
	if _, err := fmt.Sscanf(fields[0], "%d", &ahead); err != nil {
		return 0, 0, fmt.Errorf("gittree: parse ahead %q: %w", fields[0], err)
	}
	if _, err := fmt.Sscanf(fields[1], "%d", &behind); err != nil {
		return 0, 0, fmt.Errorf("gittree: parse behind %q: %w", fields[1], err)
	}
	return ahead, behind, nil
}

func remoteOf(upstream string) string {
	if i := strings.IndexByte(upstream, '/'); i > 0 {
		return upstream[:i]
	}
	return "origin"
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "\n... (truncated)"
}
