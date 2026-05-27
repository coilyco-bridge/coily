# Release pipeline

Forgejo is the canonical source for coily. GitHub is a read-only mirror.

## Flow

- Push to `main` lands on Forgejo.
- `.forgejo/workflows/release.yml` fires: tag-bump, create-release, formula-bump, windows-assets. All against Forgejo APIs.
- `.forgejo/workflows/mirror-to-github.yml` fires in parallel and on tag pushes, replicating Forgejo main and tags to GitHub via the `GITHUB_MIRROR_PAT` secret.

## Required secrets

- `FORGEJO_PAT` - Forgejo personal access token with `write:repository` scope. Used by `bump-formula` to commit the formula update via the Contents API.
- `GITHUB_MIRROR_PAT` - GitHub personal access token with `repo` scope on `coilysiren/coily`. Used by `mirror-to-github` to push main and tags to the GitHub mirror.

The mirror workflow no-ops cleanly when `GITHUB_MIRROR_PAT` is absent, so the workflow file can land before the secret is provisioned.

## Per-host remote setup

On a fresh checkout, the remote should point at Forgejo for both fetch and push. Run once per host:

```bash
git remote remove origin
git remote add origin https://forgejo.coilysiren.me/coilysiren/coily.git
git fetch origin
```

For existing clones that have the legacy bidirectional `origin` (fetch from GitHub, push to both), the same two commands flip them.

## Why

The bidirectional push was racy: the Forgejo release workflow writes new commits (tag, formula bump) on Forgejo via API; those never propagate back to GitHub via the local push path. Next push to Forgejo rejects as non-fast-forward. Forgejo-canonical + one-way mirror removes the race.

See [coily#113](https://forgejo.coilysiren.me/coilysiren/coily/issues/113).
