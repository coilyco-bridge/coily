# 11. Self-update v2 + adversarial review

Category: Incomplete features

Documented as TODO in docs/threat-model.md. Not built. Requires
decisions:

- Which second-opinion tool (codex CLI, gemini CLI, a separate Claude Code
  session)?
- How does the adversarial reviewer gate actually block a merge? GitHub
  Actions + branch protection? A custom git hook?
- Where do signed binaries live? Public GH Releases would make coily
  trivially available to anyone. Private release repo avoids that.
- How does `coily self-update` know the correct URL?

Kai explicitly deferred this on 2026-04-21.
