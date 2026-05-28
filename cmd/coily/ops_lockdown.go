package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coilysiren/cli-guard/lockdown"
	"github.com/coilysiren/cli-guard/profiles"
	"github.com/coilysiren/cli-guard/skillgen"
	"github.com/coilysiren/cli-guard/verb"
	"github.com/urfave/cli/v3"
)

// passthroughsFrontmatter is the YAML frontmatter prepended to the
// generated coily-passthroughs SKILL.md. The description is what Claude
// Code uses to decide when to load the skill.
//
//nolint:gosec // YAML frontmatter; gosec misreads the description body
const passthroughsFrontmatter = `---
name: coily-passthroughs
description: |
  Use when a shell command is denied by Claude Code's permission system
  (e.g. "Permission to use Bash with command X has been denied"), when
  reaching for aws, gh, kubectl, docker, tailscale, ssh, or scp against
  Kai's homelab, AWS account, or coilysiren resources, or when checking
  whether a privileged op has a coily wrapper. The body is a flat lookup
  table of every coily command.
---
`

// lockdownSkillCommand regenerates skills/coily-passthroughs/SKILL.md by
// walking the in-process cli.Command tree. Sits under `coily lockdown`
// because the skill is the discoverability side of the deny list - same
// event ("the coily command surface changed") regenerates both. CI
// diff-checks the file; the pre-commit hook keeps it fresh locally.
func (r *Runner) lockdownSkillCommand() *cli.Command {
	return &cli.Command{
		Name:  "skill",
		Usage: "Regenerate the coily-passthroughs skill from the in-process command tree.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "out",
				Usage: "output path (defaults to SKILL.md / commands.yaml depending on --format)",
			},
			&cli.StringFlag{
				Name:  "format",
				Usage: "markdown (default; SKILL.md for Claude Code) or yaml (structured tree for programmatic consumers)",
				Value: "markdown",
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "lockdown.skill",
				SkipScope: true,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--format": c.String("format"), "--out": c.String("out")}, nil
				},
				Action: func(_ context.Context, c *cli.Command) error {
					format := c.String("format")
					out := c.String("out")
					var body string
					switch format {
					case "markdown", "":
						body = passthroughsFrontmatter + "\n# coily passthroughs\n\n" +
							"Auto-generated lookup table of every coily verb. Regenerate with `coily lockdown skill`.\n\n" +
							"Format: full path, one-line summary, comma-separated flag names. No flag descriptions; click into `coily <path> --help` for those.\n\n" +
							skillgen.RenderMarkdown(r.builtInCommands(), "coily")
						if out == "" {
							out = "skills/coily-passthroughs/SKILL.md"
						}
					case "yaml":
						y, err := skillgen.RenderYAML(r.builtInCommands(), "coily")
						if err != nil {
							return err
						}
						body = y
						if out == "" {
							out = "skills/coily-passthroughs/commands.yaml"
						}
					default:
						return fmt.Errorf("lockdown skill: --format must be markdown or yaml, got %q", format)
					}
					if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
						return fmt.Errorf("lockdown skill: mkdir: %w", err)
					}
					if err := os.WriteFile(out, []byte(body), 0o644); err != nil {
						return fmt.Errorf("lockdown skill: write: %w", err)
					}
					fmt.Fprintln(os.Stderr, "wrote", out)
					return nil
				},
			},
			r.Audit,
		),
	}
}

// wrapperAllows maps a bare binary deny entry (the deny-list shape baked
// into cli-guard's defaults.yaml) to the explicit `Bash(coily ...:*)`
// allow that names the audited route an agent should reach for instead.
// The bare-deny is what we reject; the coily-prefixed allow is what we
// sanction. Both must ship together. Per coilysiren/coily#115.
//
// The Claude Code auto-mode classifier strips `coily tailscale status`
// back to `tailscale status` and re-applies the bare-binary deny rule
// (#159), so the explicit allow tells the classifier "this exact form
// is the audited path the deny was carved around" instead of relying on
// the bare `Bash(coily:*)` umbrella allow.
//
// Maintenance: when a new wrapped verb lands, add it here AND to
// defaults.yaml. TestWrapperAllowsParity asserts the two stay in sync.
var wrapperAllows = map[string]string{
	"Bash(tailscale:*)": "Bash(coily tailscale:*)",
	"Bash(docker:*)":    "Bash(coily docker:*)",
	"Bash(aws:*)":       "Bash(coily ops aws:*)",
	"Bash(kubectl:*)":   "Bash(coily ops kubectl:*)",
	"Bash(gh:*)":        "Bash(coily ops gh:*)",
	"Bash(flyctl:*)":    "Bash(coily ops flyctl:*)",
	"Bash(gcloud:*)":    "Bash(coily ops gcloud:*)",
	"Bash(ssh:*)":       "Bash(coily ssh:*)",
	"Bash(brew:*)":      "Bash(coily pkg brew:*)",
	"Bash(npm:*)":       "Bash(coily pkg npm:*)",
	"Bash(pnpm:*)":      "Bash(coily pkg pnpm:*)",
	"Bash(yarn:*)":      "Bash(coily pkg yarn:*)",
	"Bash(uv:*)":        "Bash(coily pkg uv:*)",
	"Bash(pip:*)":       "Bash(coily pkg pip:*)",
	"Bash(pipx:*)":      "Bash(coily pkg pipx:*)",
	"Bash(poetry:*)":    "Bash(coily pkg poetry:*)",
	"Bash(cargo:*)":     "Bash(coily pkg cargo:*)",
	"Bash(gem:*)":       "Bash(coily pkg gem:*)",
	"Bash(bundle:*)":    "Bash(coily pkg bundle:*)",
}

// applyHookHandoffTrim drops `Bash(<token>:*)` deny entries (and the
// matching `wrapperAllows` explicit allows) for every bare binary
// coily's PreToolUse hook now gates via its routing-hint table.
// The hook is the primary gate for those binaries; keeping the bare
// deny in place makes Claude Code CLI's built-in deny matcher fire
// first and clobber the hook's recovery hint, which was the audit-
// shopping pattern documented in coilysiren/coily#183.
//
// Tokens covered: keys of wrapperRecovery (declared in
// lockdown_driver.go). Tokens absent from wrapperRecovery keep their
// bare deny + their wrapperAllows explicit-allow counterweight: the
// auto-mode classifier still flags `coily ops flyctl` style wrappers
// as deny circumvention (coilysiren/coily#159) when the bare deny is
// in play, and the explicit allow tells the classifier the wrapped
// path is sanctioned.
//
// Order in the pipeline: this runs BEFORE applyWrapperAllows so the
// explicit-allow pass only counterweights surviving bare denies.
func applyHookHandoffTrim(d *lockdown.Defaults) *lockdown.Defaults {
	trimDenies := make(map[string]bool, len(wrapperRecovery))
	for token := range wrapperRecovery {
		trimDenies[fmt.Sprintf("Bash(%s:*)", token)] = true
	}
	trimAllows := make(map[string]bool, len(trimDenies))
	for deny := range trimDenies {
		if allow, ok := wrapperAllows[deny]; ok {
			trimAllows[allow] = true
		}
	}
	out := &lockdown.Defaults{
		Allow: make([]string, 0, len(d.Allow)),
		Deny:  make([]string, 0, len(d.Deny)),
	}
	for _, a := range d.Allow {
		if !trimAllows[a] {
			out.Allow = append(out.Allow, a)
		}
	}
	for _, dn := range d.Deny {
		if !trimDenies[dn] {
			out.Deny = append(out.Deny, dn)
		}
	}
	return out
}

// applyWrapperAllows augments the canonical allow list with explicit
// `Bash(coily <wrapper>:*)` entries for every bare-binary deny that
// has a sanctioned coily wrapper. See wrapperAllows. Returns a fresh
// *Defaults so the cached embedded value is not mutated.
//
// After the hook handoff (coilysiren/coily#183), this only matters
// for entries whose bare deny survives applyHookHandoffTrim
// - currently `Bash(flyctl:*)` and any future wrapped verbs not yet
// covered by coily's hook route table.
func applyWrapperAllows(d *lockdown.Defaults) *lockdown.Defaults {
	out := &lockdown.Defaults{
		Allow: append([]string(nil), d.Allow...),
		Deny:  append([]string(nil), d.Deny...),
	}
	have := make(map[string]bool, len(out.Allow))
	for _, a := range out.Allow {
		have[a] = true
	}
	for _, deny := range out.Deny {
		allow, ok := wrapperAllows[deny]
		if !ok || have[allow] {
			continue
		}
		out.Allow = append(out.Allow, allow)
		have[allow] = true
	}
	return out
}

// applyDataSecurityDenies extends the canonical deny list with extra
// entries when the lockdown driver's attached Coordinate names a high
// or max data_security tier. Phase 5 of coilysiren/coily#150.
//
// At high (and stricter): block Claude Code's Read of the coilyco-vault
// tree via the portable tilde form. Private personal context should not
// surface inside a session whose active profile names
// "data_security=high" or stricter.
//
// Returns a fresh *Defaults so the original (which the package may
// cache between calls) is not mutated.
func applyDataSecurityDenies(d *lockdown.Defaults, drv *lockdown.Driver) *lockdown.Defaults {
	if drv == nil || drv.Coordinate == nil {
		return d
	}
	tier := string(drv.Coordinate.DataSecurity)
	if tier == "" || tier == "low" || tier == "medium" {
		return d
	}
	out := &lockdown.Defaults{
		Allow: append([]string(nil), d.Allow...),
		Deny:  append([]string(nil), d.Deny...),
	}
	out.Deny = append(out.Deny, vaultReadDenies()...)
	return out
}

// vaultReadDenies returns both the portable tilde form and the
// runtime-resolved absolute form of the coilyco-vault read deny.
// Claude Code's permission matcher is a literal string compare, so
// emitting only the tilde form lets the absolute path bypass the deny.
func vaultReadDenies() []string {
	denies := []string{"Read(~/projects/coilysiren/coilyco-vault/**)"}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return denies
	}
	denies = append(denies, "Read("+home+"/projects/coilysiren/coilyco-vault/**)")
	return denies
}

// lockdownInitConfigCommand writes the embedded default profiles
// registry to ~/.coily/coily.yaml. Mirrors `coily lockdown --apply`'s
// no-clobber stance: refuses to overwrite an existing file unless
// --replace is passed. Per coilysiren/coily#150 the override file is
// the only thing that lifts any axis off Strictest, so writing it is
// an opt-in action the operator takes deliberately.
func (r *Runner) lockdownInitConfigCommand() *cli.Command {
	return &cli.Command{
		Name:  "init-config",
		Usage: "Write the embedded default profiles registry to ~/.coily/coily.yaml.",
		Description: `init-config installs the embedded default profile registry at
~/.coily/coily.yaml. The file declares the named profiles
(mobile, mac-tower, windows-laptop, web, headless) and their
per-axis tier values. coily refuses to lift any axis off the
strictest tier unless this file exists.

Refuses to overwrite an existing file by default. Pass --replace
to clobber, parallel to ` + "`coily lockdown --apply --replace`" + `.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "replace",
				Usage: "overwrite an existing ~/.coily/coily.yaml",
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "lockdown.init-config",
				SkipScope: true,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{"--replace": fmt.Sprintf("%t", c.Bool("replace"))}, nil
				},
				Action: func(_ context.Context, c *cli.Command) error {
					return lockdownInitConfigAction(c.Bool("replace"))
				},
			},
			r.Audit,
		),
	}
}

func lockdownInitConfigAction(replace bool) error {
	path, err := profiles.OverridePath()
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(path); statErr == nil && !replace {
		return fmt.Errorf("lockdown init-config: %s already exists. Use `coily lockdown init-config --replace` to overwrite", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("lockdown init-config: mkdir: %w", err)
	}
	if err := os.WriteFile(path, profiles.DefaultYAML, 0o600); err != nil {
		return fmt.Errorf("lockdown init-config: write %s: %w", path, err)
	}
	verb := "created"
	if replace {
		verb = "replaced"
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", displayPath(path), verb)
	return nil
}

// lockdownCommand is tiered by blast radius:
//
//   - bare `coily lockdown` prints the plan, no write.
//   - `coily lockdown --apply` writes only if .claude/settings.json is absent.
//     Refuses an existing file. Frictionless bootstrap.
//   - `coily lockdown --apply --replace` overwrites an existing file. This is
//     the path that can clobber custom allow/deny entries the user added by
//     hand.
//
// There is no middle ground between "bootstrap fresh" and "clobber".
func (r *Runner) lockdownCommand() *cli.Command {
	return &cli.Command{
		Name:  "lockdown",
		Usage: "Write per-repo Claude Code permissions that force all ops through coily.",
		Description: `lockdown renders a .claude/settings.json (or settings.local.json) for the
target directory with the canonical allow/deny lists baked into coily.

Three modes, by blast radius:

  coily lockdown                    Print the plan and exit. No write.
  coily lockdown --apply            Write a fresh file. Refuses if one exists.
  coily lockdown --apply --replace  Overwrite an existing settings file.

Pass --recursive to walk up to 4 directories below --path and lock down each
discovered git repo. With --apply, --recursive also merges the canonical
deny list into <path>/.claude/settings.local.json so a session started at
the recursion root cannot shadow per-repo deny rules with a broader allow.`,
		Commands: []*cli.Command{
			r.lockdownSkillCommand(),
			r.lockdownInitConfigCommand(),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "path",
				Usage: "directory whose .claude/ subdir to target",
				Value: ".",
			},
			&cli.BoolFlag{
				Name:  "local",
				Usage: "write to .claude/settings.local.json instead of settings.json",
			},
			&cli.BoolFlag{
				Name:  "apply",
				Usage: "actually write the file (default: dry-run)",
			},
			&cli.BoolFlag{
				Name:  "replace",
				Usage: "overwrite an existing settings file (requires --apply)",
			},
			&cli.BoolFlag{
				Name:  "recursive",
				Usage: "scan up to 4 directories below --path for git repos and lock down each",
			},
			&cli.BoolFlag{
				Name:  "user",
				Usage: "merge canonical denies + prune shadowed allows in ~/.claude/settings.json (exclusive with --path/--local/--recursive)",
			},
		},
		Action: r.WrapVerb(
			verb.Spec{
				Name:      "lockdown",
				SkipScope: true,
				ArgsFunc: func(c *cli.Command) (map[string]string, []string) {
					return map[string]string{
						"--path":      c.String("path"),
						"--recursive": fmt.Sprintf("%t", c.Bool("recursive")),
					}, nil
				},
				Action: lockdownAction,
			},
			r.Audit,
		),
	}
}

func lockdownAction(_ context.Context, c *cli.Command) error {
	apply := c.Bool("apply")
	replace := c.Bool("replace")
	recursive := c.Bool("recursive")
	user := c.Bool("user")

	if err := validateLockdownFlags(c, apply, replace, user, recursive); err != nil {
		return err
	}

	base, err := lockdown.LoadDefaults()
	if err != nil {
		return err
	}
	drv := coilyLockdownDriver()
	// ancestorDefaults: full deny list (no hook-handoff trim) so the
	// recursion-root settings.local.json shadow-neutralizes every
	// canonical deny. The parent has no hook to receive the handoff.
	ancestorDefaults := applyDataSecurityDenies(base, drv)
	stampDefaults := applyWrapperAllows(applyHookHandoffTrim(ancestorDefaults))

	if user {
		return lockdownUser(apply, ancestorDefaults)
	}

	root := c.String("path")
	local := c.Bool("local")

	dirs, err := lockdownTargetDirs(root, recursive)
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		if err := lockdownOne(dir, local, apply, replace, stampDefaults, drv); err != nil {
			return err
		}
	}

	if recursive {
		if err := reassertAncestor(root, apply, ancestorDefaults); err != nil {
			return err
		}
	}
	return nil
}

// validateLockdownFlags rejects illegal flag combinations early.
func validateLockdownFlags(c *cli.Command, apply, replace, user, recursive bool) error {
	if replace && !apply {
		return fmt.Errorf("lockdown: --replace requires --apply (use `coily lockdown --apply --replace`)")
	}
	if user && (recursive || c.Bool("local") || c.IsSet("path")) {
		return fmt.Errorf("lockdown: --user is exclusive with --path/--local/--recursive")
	}
	return nil
}

// lockdownTargetDirs resolves the per-repo target directories for the
// non-user lockdown modes. Single-target for plain, recursive scan otherwise.
func lockdownTargetDirs(root string, recursive bool) ([]string, error) {
	if !recursive {
		return []string{root}, nil
	}
	found, err := findGitRepos(root, recursiveScanDepth)
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("lockdown: --recursive found no git repos within %d levels of %s", recursiveScanDepth, root)
	}
	fmt.Fprintf(os.Stderr, "recursive: found %d git repo(s) under %s\n", len(found), displayPath(root))
	return found, nil
}

// lockdownUser merges canonical denies + prunes shadowed allows in
// ~/.claude/settings.json. Same MergeDenyInto semantics as the
// ancestor reassertion; targets a fixed path. See coily#128.
func lockdownUser(apply bool, d *lockdown.Defaults) error {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return fmt.Errorf("lockdown: --user: cannot resolve home directory: %w", err)
	}
	target := filepath.Join(home, ".claude", "settings.json")
	disp := displayPath(target)
	if !apply {
		fmt.Fprintf(os.Stderr, "%s: would merge canonical deny list + prune shadowed allows (--user)\n", disp)
		return nil
	}
	mutated, err := lockdown.MergeDenyInto(target, d)
	if err != nil {
		return err
	}
	if mutated {
		fmt.Fprintf(os.Stderr, "%s: merged canonical denies + pruned shadowed allows (--user)\n", disp)
	} else {
		fmt.Fprintf(os.Stderr, "%s: already covers canonical denies and has no shadowed allows (--user)\n", disp)
	}
	return nil
}

// reassertAncestor merges the canonical deny list into the recursion
// root's .claude/settings.local.json. Closes the gap surfaced by
// 2026-05-08 finding parent-dir-allowlist-overrides-per-repo-gh-lockdown:
// when Claude Code starts a session at a multi-repo parent, broad allow
// rules in the parent's settings.local.json shadow every per-repo deny
// below it. Re-asserting the deny at the parent (where Claude Code
// applies deny-before-allow within a file) neutralizes the shadow.
//
// Conservative on purpose: dry-run reports the shadow, --apply merges
// denies into the existing file and preserves any user-added allow
// entries. Never replaces or removes user content.
func reassertAncestor(root string, apply bool, d *lockdown.Defaults) error {
	target := lockdown.TargetPath(root, true)
	disp := displayPath(target)

	if !apply {
		fmt.Fprintf(os.Stderr, "%s: would merge canonical deny list (recursion-root reassertion)\n", disp)
		return nil
	}

	mutated, err := lockdown.MergeDenyInto(target, d)
	if err != nil {
		return err
	}
	if mutated {
		fmt.Fprintf(os.Stderr, "%s: merged canonical deny list (recursion-root reassertion)\n", disp)
	} else {
		fmt.Fprintf(os.Stderr, "%s: deny list already covers canonical denies (no change)\n", disp)
	}
	return nil
}

func lockdownOne(dir string, local, apply, replace bool, d *lockdown.Defaults, drv *lockdown.Driver) error {
	target := lockdown.TargetPath(dir, local)
	plan, err := lockdown.BuildPlan(target, d, drv)
	if err != nil {
		return err
	}

	disp := displayPath(plan.TargetPath)

	if !apply {
		var verb string
		switch {
		case !plan.Existed:
			verb = "would create"
		case replace:
			verb = "would replace"
		default:
			verb = "would refuse (exists; use --apply --replace to clobber)"
		}
		fmt.Fprintf(os.Stderr, "%s: %s\n", disp, verb)
		return nil
	}

	if plan.Existed && !replace {
		return fmt.Errorf("lockdown: %s already exists. Use `coily lockdown --apply --replace` to overwrite", disp)
	}

	verb := "created"
	if plan.Existed {
		verb = "replaced"
	}
	return writeLockdown(plan, d, verb)
}

// displayPath shortens an absolute path to its cwd-relative form when that
// is shorter and stays inside the working tree. Falls back to the original.
func displayPath(p string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return p
	}
	rel, err := filepath.Rel(cwd, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return p
	}
	if len(rel) < len(p) {
		return rel
	}
	return p
}

const recursiveScanDepth = 4

// findGitRepos walks root up to maxDepth levels deep looking for directories
// that contain a .git entry (file or dir, to support worktrees and submodules).
// Returns the repo directories themselves, sorted, deduplicated. The .git
// subtree is never descended into.
func findGitRepos(root string, maxDepth int) ([]string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("lockdown: resolve %s: %w", root, err)
	}
	var repos []string
	err = filepath.WalkDir(absRoot, func(path string, de os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !de.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}
		depth := 0
		if rel != "." {
			depth = len(strings.Split(rel, string(filepath.Separator)))
		}
		if depth > maxDepth {
			return filepath.SkipDir
		}
		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			repos = append(repos, path)
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("lockdown: scan %s: %w", absRoot, err)
	}
	return repos, nil
}

func writeLockdown(plan *lockdown.Plan, d *lockdown.Defaults, verb string) error {
	if err := lockdown.Write(plan); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", displayPath(plan.TargetPath), verb)
	hookPath, hookExisted, err := lockdown.WriteHook(plan.TargetPath, d, coilyLockdownDriver())
	if err != nil {
		return fmt.Errorf("lockdown: hook write failed (settings.json was written): %w", err)
	}
	hookVerb := "created"
	if hookExisted {
		hookVerb = "replaced"
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", displayPath(hookPath), hookVerb)
	return nil
}
