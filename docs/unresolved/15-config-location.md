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
