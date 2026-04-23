# 8. Embedded sub-binaries

Category: Incomplete features

Threat model says `aws/kubectl/gh` should be embedded into coily via
`//go:embed`, extracted to a per-user cache with checksum verification.
Currently `pkg/shell.PathResolver` just does exec.LookPath. The guarantee
"an agent cannot swap /usr/local/bin/aws" is not enforced. Implementation
steps:

1. Download aws-cli-v2, kubectl, gh binaries for each target platform at
   build time (or script-time).
2. `//go:embed` them with build constraints per GOOS/GOARCH.
3. Extract to `$XDG_CACHE_HOME/coily/bin/<sha256>/<tool>` on first use,
   verify sha256, exec from there.
4. Skip extraction if the cached copy checksum-matches.

Blocker: aws CLI v2 is ~30MB per platform. 4 platforms x 3 tools = 12
embedded binaries, ~100MB per final coily binary per platform. Needs
either LFS or a per-platform build + release artifact.

# Decision

Use github releases or packages ... whichever makes sense ... and overwrite the latest tag on every prod build.

If that doesn't make sure, I'll clarify. Just don't use LFS, I don't like LFS.
