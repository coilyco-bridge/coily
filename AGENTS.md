# Agent instructions

Workspace-level conventions (git workflow, test/lint autonomy, readonly ops, writing voice, deploy knowledge) are loaded globally via `~/.claude/CLAUDE.md` -> `agentic-os-kai/AGENTS.md`. This file covers only what's specific to this repo.

---

## Editing coily from any session

Edit coily from whichever session you're in. The prior cwd-gate ("forbidden from sibling sessions, file an issue instead") is retired. The global commit-msg hook (every commit closes a same-repo issue) plus the brew pipeline give the same accountability without forcing a context switch. File the issue first with `coily ops gh issue create --repo coilysiren/coily ...`, commit referencing it, push, ride the pipeline.

When sibling-repo work is blocked by a coily gap (missing subcommand, broken wrapper, policy false-positive, argv-mangling bug, deny-list miss), this is the **fix-coily-first** discipline: pivot here, make the smallest fix that unblocks the original task, commit to `main`, push, wait for the brew bump (~5 min), `brew upgrade coilysiren/tap/coily`, then return to the original repo. Every coily blocker becomes an audited coily fix, and the next op of the same shape inherits it.

**Time-critical exception:** if the original task is genuinely time-critical (live incident, deadline-bound interview reply), do the object-level work first via Kai's hands and file the coily fix as the immediate follow-up.

---

## Never bypass the release pipeline

The release-installed binary IS the contract on every host. **Do not run a locally-built coily against any real target.** Three surfaces, one rule:

- **Mac** - brew. No `go install ./cmd/coily && /Users/kai/go/bin/coily ...`, no `PATH=/Users/kai/go/bin:$PATH coily ...`, no `cp` into the Cellar.
- **Windows** - scoop. No `make install-windows` to side-step `scoop install coilysiren/coily`, no `cp` of a freshly built `bin\coily.exe` into `C:\Users\<u>\scoop\apps\coily\current\`, no `go build -o ...\coily.exe` then bare-invoke.
- **kai-server** - `make deploy-server`. No `scp` of a locally-built linux binary, no `sudo install` outside the deploy target.

The release pipeline is what `coily lockdown` trusts. Bypassing puts an unaudited binary in the path of privileged ops (ssh to kai-server, AWS calls, kubectl writes). Lockdown's argv validation and audit-log writes only fire on the release-installed binary. A locally-compiled copy passes the same name check but has whatever local source you just compiled, including unreviewed changes.

This is a security boundary, not a hygiene preference. Mechanical denies for the Mac shape live in `~/.claude/settings.json` (`Bash(*go/bin/coily*)`, `Bash(PATH=*coily*)`, etc.). The Windows shape needs its own denies (`*scoop\apps\coily\current\*` writes, `*\bin\coily.exe` bare invokes from a repo checkout). This rule covers the broader pattern those denies don't catch.

If you find yourself reaching for a `PATH=` prepend, a `cp` into the Cellar, or a `make install-windows` to make a tool work right now, **stop**. The right move is either (a) wait for the release, (b) work around the missing capability through a different tool, or (c) ship a smaller, faster change that goes through the pipeline.

---

## Default to `coily ops gh`, not raw `gh`

For any GitHub op in Kai's workspace - `gh api`, `gh issue`, `gh pr`, `gh repo`, `gh search`, `gh run`, `gh workflow`, `gh release`, `gh secret` - reach for `coily ops gh ...` first. Same flags, same behavior, just routed through the wrapper so it gets audit-logged and obeys the lockdown deny list.

`coily ops gh api` flag note: it's `--jq`, not `-q`. Otherwise the surface mirrors `gh` directly.

Raw `gh` is the fallback only when coily isn't installed (CI, fresh machine pre-bootstrap). In an interactive session on Kai's hosts, `coily ops gh` is the default.

---

## Release framework

Every push to `main` triggers `.github/workflows/release.yml`, which fully automates versioning and Homebrew distribution. No manual `make release`, no manual tag, no manual PR.

**Per-push flow:**

1. `mathieudutour/github-tag-action` computes the next semver from commits since the last tag. `default_bump: patch` means *every* push releases at least a patch.
   - plain commit -> patch bump
   - `feat: ...` -> minor bump
   - `feat!: ...` or body containing `BREAKING CHANGE:` -> major bump
2. The new tag is created and pushed. Unlike repo-recall, there is no source-tree version bump: `main.Version` is set at build time via ldflags (brew formula on Mac, the windows-build job on Windows).
3. A GitHub Release is cut with the auto-generated changelog.
4. Two consumers fan out in parallel from the new tag:
   - `bump-tap` downloads the new tarball, computes its sha256, and pushes the updated Formula directly to `main` on `coilysiren/homebrew-tap`. No PR. Brew builds from source on `brew upgrade`.
   - `windows-build` cross-compiles `coily-windows-{amd64,arm64}.exe` and a matching `.sha256` sidecar, attaches both to the release as assets. `coilysiren/scoop-bucket` autoupdates from those URLs - `scoop update coily` pulls the new binary directly, no source compile.

**Loop safety:** the tag is created by an action using `GITHUB_TOKEN`, which by GitHub policy doesn't re-trigger workflows. So the release job won't recurse on its own tag.

**Secret required:** `HOMEBREW_TAP_TOKEN` - fine-grained PAT scoped to `coilysiren/homebrew-tap` with `Repository permissions -> Contents: Read and write`. Set via `gh secret set HOMEBREW_TAP_TOKEN --repo coilysiren/coily`.

**Formula source of truth:** `Formula/coily.rb` in `coilysiren/homebrew-tap`. The bump-tap job edits `url` + `sha256` only - any structural change to the formula (new dependency, install steps, test block) is a hand-edit on the tap and is not overwritten on release.

**Scoop manifest source of truth:** `bucket/coily.json` in `coilysiren/scoop-bucket`. The autoupdate block points at `https://github.com/coilysiren/coily/releases/download/v$version/coily-windows-<arch>.exe#/coily.exe` and reads SHA256 from the `.sha256` sidecar. Structural changes (new architectures, post_install hooks) are hand-edits on the bucket and survive autoupdate.

**Install-path caveat (brew / scoop):** both managed installs produce *user-writable* binaries - `/opt/homebrew/bin/coily` on Mac, `C:\Users\<u>\scoop\apps\coily\current\coily.exe` on Windows. Both lose the admin-owned-binary property that `make install` (unix) and `make install-windows` (Program Files) preserved. Brew and scoop are for fresh-machine bootstrap and the auto-upgrade loop. The hardened install path (root-owned `/usr/local/bin/coily` or `C:\Program Files\coily\coily.exe`) is a manual choice for hosts that need it.

---

## Post-push follow-up (auto-schedule)

Per the workspace "Default to proactive scheduling" rule: after pushing to `main`, schedule a wake-up to land the new binary on Kai's laptop and re-baseline the lockdown rules. The release workflow needs ~1-3 min to finish (tag + GitHub Release + tap formula push).

- **Cadence**: 300-360s after push. Cache stays warm at 270s but the tap-bump can lag past that, so 300s is the floor. The windows-build job finishes within the same window.
- **Verify CI green first**: `coily ops gh run list --repo coilysiren/coily --limit 1` should show the release run as `completed/success`. If still in progress, re-schedule once at +180s. If failed, surface the failure and stop.
- **Upgrade on the active host**:
  - **Mac** - `brew outdated coilysiren/tap/coily` and, if upgradeable, `brew upgrade coilysiren/tap/coily`. No sudo (Homebrew installs to user-writable `/opt/homebrew`).
  - **Windows** - `scoop update coily`. The autoupdate block in `coilysiren/scoop-bucket/bucket/coily.json` reads the new version from the latest GitHub release and the hash from the `.sha256` sidecar. No admin (scoop installs to `~/scoop/apps/coily/`).
- **Re-baseline lockdown** *only when the bumped commit changed `pkg/lockdown/` or `Formula/coily.rb`-relevant code*: `coily lockdown --apply --replace --recursive --path ~/projects/coilysiren`. Skip when the bump is unrelated to lockdown defaults (most patch bumps).
- **Trigger kai-server update**: `coily ssh kai-server -- coily systemctl start coily-update.service`. The unit runs `brew upgrade coilysiren/tap/coily` + `coily setup` on the server with `COILY_LOCKDOWN_ROOT=/home/kai/projects/coilysiren`. Oneshot returns immediately. Verify with `coily ssh kai-server -- coily systemctl status coily-update.service`. Skip when the bump is laptop-only (macOS-specific code path, Windows-only fix, etc.).
- **Skip the auto-schedule** when the push is documentation-only (README, AGENTS.md, docs/) - the binary changes but nothing in it has user-visible effect.

---

## Commands

Route every dev command through coily, which reads [`.coily/coily.yaml`](.coily/coily.yaml). Even in this repo, the lockdown denies bare invocations of the underlying tools (`make`, `go`, etc.). Add new verbs to that file before invoking them.

## See also

- [README.md](README.md) - human-facing intro.
- [docs/FEATURES.md](docs/FEATURES.md) - inventory of what ships today.
- [.coily/coily.yaml](.coily/coily.yaml) - allowlisted commands. Agents route through coily, not bare `make` / `uv` / `python` / `npm` / `cargo` / `dotnet`.

Cross-reference convention from [coilysiren/agentic-os-kai#313](https://github.com/coilysiren/agentic-os-kai/issues/313).
