---
name: coily-passthroughs
description: |
  Use when a shell command is denied by Claude Code's permission system
  (e.g. "Permission to use Bash with command X has been denied"), when
  reaching for aws, gh, kubectl, docker, tailscale, ssh, or scp against
  Kai's homelab, AWS account, or coilysiren resources, or when checking
  whether a privileged op has a coily wrapper. The body is a flat lookup
  table of every coily command.
---

# coily passthroughs

Auto-generated lookup table of every coily verb. Regenerate with `coily lockdown skill`.

Format: full path, one-line summary, comma-separated flag names. No flag descriptions; click into `coily <path> --help` for those.

## `coily audit finding`

Walk an agent through writing a finding about a flagged audit event.

Flags: --id, --slug, --ts, --verb

## `coily audit path`

Print the resolved audit log path and exit.

## `coily audit tail`

Stream audit records as JSONL.

Flags: --follow, --since

## `coily docker`

Pass-through to docker with argv validation + audit log.

## `coily gaming core-keeper restart`

Restart the core-keeper-server unit.

## `coily gaming core-keeper start`

Start the core-keeper-server unit.

## `coily gaming core-keeper status`

Print systemctl status core-keeper-server.

## `coily gaming core-keeper stop`

Stop the core-keeper-server unit.

## `coily gaming core-keeper tail`

Tail core-keeper-server journal logs (journalctl -u core-keeper-server -f).

Flags: --follow, --lines

## `coily gaming eco mod push`

scp a .zip to <server_dir> on kai-server and unzip -o it.

Flags: --keep-remote, --server-dir, --src

## `coily gaming eco restart`

Restart the eco-server systemd unit.

## `coily gaming eco start`

Start the eco-server systemd unit.

## `coily gaming eco status`

Print systemctl status eco-server.

## `coily gaming eco stop`

Stop the eco-server systemd unit.

## `coily gaming eco tail`

Tail eco-server journal logs (journalctl -u eco-server -f).

Flags: --follow, --lines

## `coily gaming eco world get-seed`

Print the current Seed from Configs/WorldGenerator.eco.

Flags: --configs-dir

## `coily gaming eco world randomize`

Generate a random seed and write it to Configs/WorldGenerator.eco.

Flags: --configs-dir

## `coily gaming eco world set-seed`

Write a specific Seed into Configs/WorldGenerator.eco.

Flags: --configs-dir, --seed

## `coily gaming eco world snapshot`

Copy Configs/WorldGenerator.eco to --target.

Flags: --configs-dir, --target

## `coily gaming factorio mods list`

Print mod-list.json entries with their enabled flag.

## `coily gaming factorio mods sync`

Pull the mod files in mods/ into agreement with mod-list.json.

Flags: --dry-run, --mod

## `coily gaming factorio players adminlist`

Print entries from server-adminlist.json.

## `coily gaming factorio players banlist`

Print entries from server-banlist.json.

## `coily gaming factorio players whitelist`

Print entries from server-whitelist.json.

## `coily gaming factorio restart`

Restart the factorio-server unit.

## `coily gaming factorio saves backup-now`

Trigger an immediate off-cluster snapshot of the saves dir.

## `coily gaming factorio saves list`

List zip saves under the FactorioServer/saves directory.

## `coily gaming factorio start`

Start the factorio-server unit.

## `coily gaming factorio status`

Print systemctl status factorio-server.

## `coily gaming factorio stop`

Stop the factorio-server unit.

## `coily gaming factorio tail`

Tail factorio-server journal logs (journalctl -u factorio-server -f).

Flags: --follow, --lines

## `coily gaming factorio update`

Run steamcmd against app 427520 to update the factorio install.

## `coily gaming icarus restart`

Restart the icarus-server unit.

## `coily gaming icarus start`

Start the icarus-server unit.

## `coily gaming icarus status`

Print systemctl status icarus-server.

## `coily gaming icarus stop`

Stop the icarus-server unit.

## `coily gaming icarus tail`

Tail icarus-server journal logs (journalctl -u icarus-server -f).

Flags: --follow, --lines

## `coily git audit-show`

Resolve an Audit-log trailer back to its full audit record.

Flags: --scope, --since

## `coily git trailer`

Emit Audit-log: trailers for the current repo.

Flags: --max, --scope, --since

## `coily git trailer-hook`

prepare-commit-msg hook: append Audit-log: trailers in place.

## `coily glama attributes`

GET /v1/attributes - the curated attribute vocabulary.

## `coily glama instances`

GET /v1/instances - hosted MCP server instances (bearer auth).

## `coily glama servers get`

GET /v1/servers/{namespace}/{slug}

## `coily glama servers list`

GET /v1/servers - paginated directory listing.

Flags: --after, --first, --query

## `coily glama telemetry usage`

POST /v1/telemetry/usage - send tool-usage data.

Flags: --body

## `coily install-completion`

Install shell tab-completion for coily.

Flags: --dry-run, --shell

## `coily lockdown skill`

Regenerate the coily-passthroughs skill from the in-process command tree.

Flags: --format, --out

## `coily modio mods comments`

GET /games/{game-id}/mods/{mod-id}/comments

## `coily modio mods files`

GET /games/{game-id}/mods/{mod-id}/files

## `coily modio mods get`

GET /games/{game-id}/mods/{mod-id}

## `coily modio mods list`

GET /games/{game-id}/mods

Flags: --limit, --offset

## `coily ops aws`

Pass-through to aws with argv validation + audit log.

## `coily ops gh`

Pass-through to gh with argv validation + audit log.

## `coily ops kubectl`

Pass-through to kubectl with argv validation + audit log.

## `coily pkg brew`

Pass-through to brew with argv validation + audit log.

## `coily pkg bun`

Pass-through to bun with argv validation + audit log.

## `coily pkg bundle`

Pass-through to bundle with argv validation + audit log.

## `coily pkg cargo`

Pass-through to cargo with argv validation + audit log.

## `coily pkg gem`

Pass-through to gem with argv validation + audit log.

## `coily pkg npm`

Pass-through to npm with argv validation + audit log.

## `coily pkg pip`

Pass-through to pip with argv validation + audit log.

## `coily pkg pipx`

Pass-through to pipx with argv validation + audit log.

## `coily pkg pnpm`

Pass-through to pnpm with argv validation + audit log.

## `coily pkg poetry`

Pass-through to poetry with argv validation + audit log.

## `coily pkg uv`

Pass-through to uv with argv validation + audit log.

## `coily pkg yarn`

Pass-through to yarn with argv validation + audit log.

## `coily setup`

Run the post-upgrade rituals: completion, lockdown re-baseline, and user hook.

Flags: --skip-completion, --skip-lockdown, --skip-user-hook, --workspace

## `coily sirens-discord-ops restart`

Restart the sirens-discord-ops systemd unit.

## `coily sirens-discord-ops start`

Start the sirens-discord-ops systemd unit.

## `coily sirens-discord-ops status`

Print systemctl status sirens-discord-ops.

## `coily sirens-discord-ops stop`

Stop the sirens-discord-ops systemd unit.

## `coily sirens-discord-ops tail`

Tail sirens-discord-ops journal logs (journalctl -u sirens-discord-ops -f).

Flags: --follow, --lines

## `coily skillsmp ai-search`

GET /ai-search - semantic search across skills.

## `coily skillsmp search`

GET /search - keyword search across skills.

Flags: --limit, --page, --sort-by

## `coily ssh cat`

Run cat <path>.

Flags: --host, --user

## `coily ssh copy`

Upload a local file to the remote via sftp.

Flags: --host, --user

## `coily ssh deploy eco-mod`

Fetch the latest release zip(s) of <name> and unzip into the EcoServer tree.

Flags: --host, --user

## `coily ssh deploy eco-mod-source`

Rsync the source-tree Eco mod <name> from eco-mods / eco-mods-public into the EcoServer tree.

Flags: --host, --user

## `coily ssh deploy repo-recall`

Fast-forward /home/kai/projects/coilysiren/repo-recall and run /home/kai/projects/coilysiren/infrastructure/scripts/install-repo-recall.sh as root.

Flags: --host, --user

## `coily ssh file`

Run file <path>.

Flags: --host, --user

## `coily ssh git branch`

Run git branch --show-current in <repo-path>.

Flags: --host, --user

## `coily ssh git diff`

Run git diff in <repo-path> (read-only, unstaged changes).

Flags: --host, --user

## `coily ssh git fetch`

Run git fetch --all --prune in <repo-path>.

Flags: --host, --user

## `coily ssh git log`

Run git log --oneline -n 20 in <repo-path>.

Flags: --host, --user

## `coily ssh git pull`

Run git pull --ff-only in <repo-path>.

Flags: --host, --user

## `coily ssh git rev-parse`

Run git rev-parse HEAD in <repo-path>.

Flags: --host, --user

## `coily ssh git status`

Run git status --short --branch in <repo-path>.

Flags: --host, --user

## `coily ssh git update-index`

Run git update-index --really-refresh in <repo-path> (re-hashes file contents to clear stat-stale phantom-dirty entries; mutates only the index's cached-stat column).

Flags: --host, --user

## `coily ssh grep`

Run grep -F -- '<pattern>' <path> (fixed-string match).

Flags: --host, --user

## `coily ssh head`

Run head <path> (first 10 lines).

Flags: --host, --user

## `coily ssh journalctl`

Run journalctl -u <unit> -n <lines> --no-pager on the remote.

Flags: --host, --lines, --user

## `coily ssh kubectl`

Run `sudo k3s kubectl <args>` on kai-server.

## `coily ssh ls`

Run ls -la <path>.

Flags: --host, --user

## `coily ssh rm-unit`

Remove /etc/systemd/system/<unit>.service and reload systemd.

Flags: --host, --user

## `coily ssh systemctl daemon-reload`

Run systemctl daemon-reload.

Flags: --host, --user

## `coily ssh systemctl disable`

Disable <unit>.

Flags: --host, --user

## `coily ssh systemctl enable`

Enable <unit>.

Flags: --host, --user

## `coily ssh systemctl restart`

Restart <unit>.

Flags: --host, --user

## `coily ssh systemctl start`

Start <unit>.

Flags: --host, --user

## `coily ssh systemctl status`

Print systemctl status of <unit>.

Flags: --host, --user

## `coily ssh systemctl stop`

Stop <unit>.

Flags: --host, --user

## `coily ssh tail`

Run tail <path> (last 10 lines).

Flags: --host, --user

## `coily ssh tree`

Run tree -L 2 <path> (depth-limited).

Flags: --host, --user

## `coily ssh wc`

Run wc <path>.

Flags: --host, --user

## `coily tailscale`

Pass-through to tailscale with argv validation + audit log.

## `coily trello create`

Create a new Trello card.

Flags: --desc, --dir, --label, --list, --name

## `coily trello sort`

Sort each list so cards with the given label rise to the top.

Flags: --dir, --dry, --label

## `coily trello status`

List Trello cards with their list, labels, and last activity.

Flags: --dir

## `coily trello update`

Mutate one Trello card (move list, toggle labels, append a comment, rename, close/reopen).

Flags: --close, --comment, --desc, --dir, --label-off, --label-on, --list, --name, --reopen

## `coily version`

Print the build version and exit.

## `coily whoami`

Print the authenticated identity coily sees across aws, kubectl, and gh.
