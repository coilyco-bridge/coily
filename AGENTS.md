# Agent instructions

Workspace conventions load globally via `~/.claude/CLAUDE.md` -> `agentic-os-kai/AGENTS.md`. This file covers only what's specific to this repo.

---

## Editing coily from any session

Edit coily from whichever session you're in. The commit-msg hook (every commit closes a same-repo issue) plus the brew pipeline carry the accountability. File the issue, commit referencing it, push, ride the pipeline.

When sibling-repo work is blocked by a coily gap, **fix-coily-first**: pivot here, smallest fix that unblocks, commit + push, wait for the brew bump (~5 min), `brew upgrade coilysiren/tap/coily`, return to the original repo. Each blocker becomes an audited fix; the next op of the same shape inherits it. Time-critical exception: live incidents or deadline replies get the object-level work first via Kai's hands, coily fix as immediate follow-up.

## Never bypass the release pipeline

The release-installed binary IS the contract on every host. **Do not run a locally-built coily against any real target.**

- **Mac** - brew only. No `go install ./cmd/coily && /Users/kai/go/bin/coily`, no `PATH=/Users/kai/go/bin:$PATH coily`, no `cp` into the Cellar.
- **Windows** - scoop only. No `make install-windows` to side-step scoop, no `cp` of a freshly built `bin\coily.exe` into the scoop apps dir.
- **kai-server** - `make deploy-server` only. No `scp` of a local binary, no `sudo install` outside the deploy target.

The release pipeline is what `coily lockdown` trusts. Bypassing puts an unaudited binary in the path of privileged ops. Argv validation and audit-log writes only fire on the release-installed binary. Mechanical denies for known shapes live in `~/.claude/settings.json`; this rule covers the broader pattern those denies don't catch. If you reach for a `PATH=` prepend, a `cp` into the Cellar, or a side-step install, **stop**. Either wait for the release, work around through a different tool, or ship a smaller faster change through the pipeline.

## Default to `coily ops gh`, not raw `gh`

For any GitHub op in Kai's workspace, reach for `coily ops gh ...` first. Same flags, same behavior, routed through audit + lockdown. Note: `--jq`, not `-q`. Raw `gh` is the fallback only when coily isn't installed (CI, fresh machine pre-bootstrap).

## Release framework

Every push to `main` triggers `.github/workflows/release.yml`, fully automated.

1. `mathieudutour/github-tag-action` computes the next semver. `default_bump: patch`: every push releases at least a patch. `feat:` -> minor, `feat!:` / `BREAKING CHANGE:` -> major.
2. Tag pushed. `main.Version` set at build time via ldflags (brew formula / windows-build job); no source-tree version bump.
3. GitHub Release with auto-changelog.
4. Two consumers fan out: `bump-tap` pushes updated `Formula/coily.rb` to `coilysiren/homebrew-tap` (brew builds from source on `brew upgrade`); `windows-build` cross-compiles + uploads `coily-windows-{amd64,arm64}.exe` + `.sha256` sidecars, `coilysiren/scoop-bucket` autoupdates from those URLs.

Loop-safe: `GITHUB_TOKEN`-created tags don't re-trigger workflows. Secret required: `HOMEBREW_TAP_TOKEN` fine-grained PAT scoped to `coilysiren/homebrew-tap` with Contents: Read and write. Formula and Scoop manifest are sources of truth on their respective tap/bucket repos; bump jobs touch only `url` / `sha256`.

Brew + scoop produce user-writable binaries (`/opt/homebrew/bin/coily`, `~/scoop/apps/coily/current/coily.exe`). The root-owned property of `make install` / `make install-windows` is the manual choice for hosts that need it.

## Post-push follow-up (auto-schedule)

After pushing to `main`, schedule a wake-up to land the new binary and re-baseline lockdown. Release pipeline takes ~1-3 min.

- **Cadence**: 300-360s after push.
- **Verify CI**: `coily ops gh run list --repo coilysiren/coily --limit 1` should be `completed/success`. Re-schedule once at +180s if in progress; stop on failure.
- **Upgrade host**: Mac `brew upgrade coilysiren/tap/coily`. Windows `scoop update coily`. No sudo on either.
- **Re-baseline lockdown** only when the bumped commit touched `cli-guard/lockdown/`: `coily lockdown --apply --replace --recursive --path ~/projects/coilysiren`.
- **kai-server**: `coily ssh kai-server -- coily systemctl start coily-update.service`. Oneshot.
- **Skip** for docs-only pushes.

## Commands

Route every dev command through coily ([`.coily/coily.yaml`](.coily/coily.yaml)). Lockdown denies bare invocations of underlying tools (`make`, `go`, etc.). Add new verbs to the YAML before invoking.

## See also

- [README.md](README.md) - human-facing intro.
- [docs/FEATURES.md](docs/FEATURES.md) - inventory of what ships today.
- [.coily/coily.yaml](.coily/coily.yaml) - allowlisted commands.

Cross-reference convention from [coilysiren/agentic-os#59](https://github.com/coilysiren/agentic-os/issues/59).
