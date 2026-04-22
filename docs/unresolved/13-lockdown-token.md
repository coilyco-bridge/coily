# 13. Should `coily lockdown` require a token?

Category: Open questions

Currently ReadOnly. It writes local files (`.claude/settings.json`), which
feels like a write operation. But the file it writes is the safety
boundary, and requiring a token to turn on the boundary is awkward. My
choice was ReadOnly + require `--apply` as the explicit gate. Reconsider
if an agent is ever observed running `coily lockdown --apply --replace`
in an unexpected context.
