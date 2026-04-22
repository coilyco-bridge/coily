# 6. Audit log perms are 0600 but the default path is ~/.local/state

Category: Known bugs and rough edges

If the config specifies `/var/log/coily/audit.jsonl` coily will fail to
write because that path needs root. Runtime silently falls back to the
configured path and just fails the write with stderr warnings per call.
Consider: either make the fallback loud, or default the config to
`~/.local/state/coily/audit.jsonl`. Right now `config.yaml` uses
`~/.local/state/coily/audit.jsonl` which is correct for personal laptop
use but not for a multi-user deploy.
