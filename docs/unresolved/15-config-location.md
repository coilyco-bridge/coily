# 15. Where does the coily config actually live?

Category: Open questions

Right now the config.yaml is committed in the repo root (gitignored at
/config.yaml, separately committed as /config.example.yaml). The Makefile
copies it into pkg/config/ before build. Kai's personal copy carries
paths like `~/.local/state/coily/audit.jsonl`.

If coily is ever installed on kai-server (linux ARM), should the same
config apply? Or does the server need its own config? Current Makefile
target `make deploy-server` carries Kai's laptop config over to the
server verbatim. Might want a `make deploy-server CONFIG=server.yaml`
override.

# Decision

Globals go in `~/.coily/`. Locals go in `./.coily/` (a `.coily` folder in the repo root). Even the simpler `config.yaml` files.

## Precedence

Local overrides global. When both `~/.coily/config.yaml` and `./.coily/config.yaml` exist, local wins.

## What's global vs local

- **Audits** - global. They need to be durable and live beyond the life of the repo, but nested in a folder named after the repo. Another doc describes this.
- **Tokens / credentials** - global. Same reasoning as audits.
- **Allowlists** - local. Live in `./.coily/`.

## Makefile

Repos should carry as little Makefile as possible. Just enough to set up coily (and even that may not be needed). Tests and other tooling live inside `./.coily/`.

## Server

Same rules apply. The server does not hold canonical versions of its own configs. Push from a local repo into the server. The server's configs are git-tracked on the laptop side, not authored on the server.
