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

## `coily audit path`

Print the resolved audit log path and exit.

## `coily audit tail`

Stream audit records as JSONL.

Flags: --follow, --since

## `coily aws`

Pass-through to aws with argv validation + audit log.

## `coily brew`

Pass-through to brew with argv validation + audit log.

## `coily bun`

Pass-through to bun with argv validation + audit log.

## `coily bundle`

Pass-through to bundle with argv validation + audit log.

## `coily cargo`

Pass-through to cargo with argv validation + audit log.

## `coily core-keeper restart`

Restart the core-keeper-server unit.

## `coily core-keeper start`

Start the core-keeper-server unit.

## `coily core-keeper status`

Print systemctl status core-keeper-server.

## `coily core-keeper stop`

Stop the core-keeper-server unit.

## `coily core-keeper tail`

Tail core-keeper-server journal logs (journalctl -u core-keeper-server -f).

Flags: --follow, --lines

## `coily docker`

Pass-through to docker with argv validation + audit log.

## `coily eco mod push`

scp a .zip to <server_dir> on kai-server and unzip -o it.

Flags: --keep-remote, --server-dir, --src

## `coily eco restart`

Restart the eco-server systemd unit.

## `coily eco start`

Start the eco-server systemd unit.

## `coily eco status`

Print systemctl status eco-server.

## `coily eco stop`

Stop the eco-server systemd unit.

## `coily eco tail`

Tail eco-server journal logs (journalctl -u eco-server -f).

Flags: --follow, --lines

## `coily eco world get-seed`

Print the current Seed from Configs/WorldGenerator.eco.

Flags: --configs-dir

## `coily eco world randomize`

Generate a random seed and write it to Configs/WorldGenerator.eco.

Flags: --configs-dir

## `coily eco world set-seed`

Write a specific Seed into Configs/WorldGenerator.eco.

Flags: --configs-dir, --seed

## `coily eco world snapshot`

Copy Configs/WorldGenerator.eco to --target.

Flags: --configs-dir, --target

## `coily gem`

Pass-through to gem with argv validation + audit log.

## `coily gh`

Pass-through to gh with argv validation + audit log.

## `coily icarus restart`

Restart the icarus-server unit.

## `coily icarus start`

Start the icarus-server unit.

## `coily icarus status`

Print systemctl status icarus-server.

## `coily icarus stop`

Stop the icarus-server unit.

## `coily icarus tail`

Tail icarus-server journal logs (journalctl -u icarus-server -f).

Flags: --follow, --lines

## `coily install-completion`

Install shell tab-completion for coily.

Flags: --dry-run, --shell

## `coily kubectl`

Pass-through to kubectl with argv validation + audit log.

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

## `coily npm`

Pass-through to npm with argv validation + audit log.

## `coily pip`

Pass-through to pip with argv validation + audit log.

## `coily pipx`

Pass-through to pipx with argv validation + audit log.

## `coily pnpm`

Pass-through to pnpm with argv validation + audit log.

## `coily poetry`

Pass-through to poetry with argv validation + audit log.

## `coily setup`

Run the post-upgrade rituals: completion, skill symlink, and lockdown re-baseline.

Flags: --skip-completion, --skip-lockdown, --skip-skill, --workspace

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

## `coily ssh grep`

Run grep -F -- '<pattern>' <path> (fixed-string match).

Flags: --host, --user

## `coily ssh head`

Run head <path> (first 10 lines).

Flags: --host, --user

## `coily ssh journalctl`

Run journalctl -u <unit> -n <lines> --no-pager on the remote.

Flags: --host, --lines, --user

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

## `coily uv`

Pass-through to uv with argv validation + audit log.

## `coily version`

Print the build version and exit.

## `coily whoami`

Print the authenticated identity coily sees across aws, kubectl, and gh.

## `coily yarn`

Pass-through to yarn with argv validation + audit log.
