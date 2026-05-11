# pkg/

Reusable, coily-agnostic Go packages. Treat this tree as if it were already
a separate Go module: do not regress the properties that make a future split
mechanical rather than archaeological.

Rules for new code under `pkg/`:

- **No `cmd/coily` imports.** The import graph is one-way: `cmd/` may
  import `pkg/`, never the reverse. Sibling `pkg/` packages may import
  each other only along the existing edges; new cross-package edges
  should survive a "would this make sense in a different binary" sniff.
- **No coily-specific defaults baked into exported structs.** Defaults
  may exist (see `pkg/lockdown/defaults.yaml`), but they should be
  loaded explicitly rather than implicit in a zero value.
- **Godoc on every exported name.** Read it cold and check that the
  surface tells a coherent story without coily-as-driver context. The
  `pkg/exitcode` and `pkg/workdir` package comments are the model.
- **No `cmd/coily` types leaking down into `pkg/` signatures.** If a
  helper needs a coily-shaped argument, define the type in `pkg/`
  and have `cmd/coily` adapt to it, not the other way around.

Why not split the repo today: see [issue #69](https://github.com/coilysiren/coily/issues/69).
The short version is that library APIs only firm up under second-consumer
pressure, and the cost of a split (semver, breaking-change discipline,
cross-cutting PRs in lockstep) buys nothing until another caller exists.
The rules above capture most of the upside without paying the split cost.
