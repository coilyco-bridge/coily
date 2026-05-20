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

## `coily audit dashboard`

Render a static HTML aggregation of the audit log.

Flags: --out, --since

## `coily audit finding`

Walk an agent through filing a finding GitHub issue about a flagged audit event.

Flags: --id, --slug, --ts, --verb

## `coily audit open`

Open the rendered audit dashboard in the default browser.

Flags: --path

## `coily audit path`

Print the resolved audit log path and exit.

## `coily audit tail`

Stream audit records as JSONL.

Flags: --follow, --since

## `coily dispatch headless`

Fire `claude -p` non-interactively against a real open coilysiren/* issue.

Flags: --allowed-tools, --claude-bin, --dry-run, --permission-mode

## `coily dispatch interactive`

Open a new Warp tab with `claude "Work on issue <ref>"` pre-submitted.

Flags: --dry-run, --launch-name, --scratch-path

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

## `coily hook pre-tool-use`

PreToolUse hook for the Bash tool. Routes bare-binary invocations through coily wrappers with a recovery hint; rejects coily-binary invocations resolving outside the canonical install paths.

## `coily install-completion`

Install shell tab-completion for coily.

Flags: --dry-run, --shell

## `coily lint`

Lint .coily/coily.yaml against the repo Makefile.

## `coily lockdown init-config`

Write the embedded default profiles registry to ~/.coily/coily.yaml.

Flags: --replace

## `coily lockdown skill`

Regenerate the coily-passthroughs skill from the in-process command tree.

Flags: --format, --out

## `coily ops aws`

Pass-through to aws with argv validation + audit log.

## `coily ops claude-remote-control restart`

Restart the claude-remote-control unit.

## `coily ops claude-remote-control start`

Start the claude-remote-control unit.

## `coily ops claude-remote-control status`

Print systemctl status claude-remote-control.

## `coily ops claude-remote-control stop`

Stop the claude-remote-control unit.

## `coily ops claude-remote-control tail`

Tail claude-remote-control journal logs (journalctl -u claude-remote-control -f).

Flags: --follow, --lines

## `coily ops discord applications applications-get-activity-instance`

GET /applications/{application_id}/activity-instances/{instance_id}

## `coily ops discord applications bulk-set-application-commands`

PUT /applications/{application_id}/commands

Flags: --body

## `coily ops discord applications bulk-set-guild-application-commands`

PUT /applications/{application_id}/guilds/{guild_id}/commands

Flags: --body

## `coily ops discord applications consume-entitlement`

POST /applications/{application_id}/entitlements/{entitlement_id}/consume

## `coily ops discord applications create-application-command`

POST /applications/{application_id}/commands

Flags: --body

## `coily ops discord applications create-application-emoji`

POST /applications/{application_id}/emojis

Flags: --body

## `coily ops discord applications create-entitlement`

POST /applications/{application_id}/entitlements

Flags: --body

## `coily ops discord applications create-guild-application-command`

POST /applications/{application_id}/guilds/{guild_id}/commands

Flags: --body

## `coily ops discord applications delete-application-command`

DELETE /applications/{application_id}/commands/{command_id}

## `coily ops discord applications delete-application-emoji`

DELETE /applications/{application_id}/emojis/{emoji_id}

## `coily ops discord applications delete-entitlement`

DELETE /applications/{application_id}/entitlements/{entitlement_id}

## `coily ops discord applications delete-guild-application-command`

DELETE /applications/{application_id}/guilds/{guild_id}/commands/{command_id}

## `coily ops discord applications get-application`

GET /applications/{application_id}

## `coily ops discord applications get-application-command`

GET /applications/{application_id}/commands/{command_id}

## `coily ops discord applications get-application-emoji`

GET /applications/{application_id}/emojis/{emoji_id}

## `coily ops discord applications get-application-role-connections-metadata`

GET /applications/{application_id}/role-connections/metadata

## `coily ops discord applications get-entitlement`

GET /applications/{application_id}/entitlements/{entitlement_id}

## `coily ops discord applications get-entitlements`

GET /applications/{application_id}/entitlements

Flags: --after, --before, --exclude_deleted, --exclude_ended, --guild_id, --limit, --only_active, --sku_ids, --user_id

## `coily ops discord applications get-guild-application-command`

GET /applications/{application_id}/guilds/{guild_id}/commands/{command_id}

## `coily ops discord applications get-guild-application-command-permissions`

GET /applications/{application_id}/guilds/{guild_id}/commands/{command_id}/permissions

## `coily ops discord applications get-my-application`

GET /applications/@me

## `coily ops discord applications list-application-commands`

GET /applications/{application_id}/commands

Flags: --with_localizations

## `coily ops discord applications list-application-emojis`

GET /applications/{application_id}/emojis

## `coily ops discord applications list-guild-application-command-permissions`

GET /applications/{application_id}/guilds/{guild_id}/commands/permissions

## `coily ops discord applications list-guild-application-commands`

GET /applications/{application_id}/guilds/{guild_id}/commands

Flags: --with_localizations

## `coily ops discord applications set-guild-application-command-permissions`

PUT /applications/{application_id}/guilds/{guild_id}/commands/{command_id}/permissions

Flags: --body

## `coily ops discord applications update-application`

PATCH /applications/{application_id}

Flags: --body

## `coily ops discord applications update-application-command`

PATCH /applications/{application_id}/commands/{command_id}

Flags: --body

## `coily ops discord applications update-application-emoji`

PATCH /applications/{application_id}/emojis/{emoji_id}

Flags: --body

## `coily ops discord applications update-application-role-connections-metadata`

PUT /applications/{application_id}/role-connections/metadata

Flags: --body

## `coily ops discord applications update-guild-application-command`

PATCH /applications/{application_id}/guilds/{guild_id}/commands/{command_id}

Flags: --body

## `coily ops discord applications update-my-application`

PATCH /applications/@me

Flags: --body

## `coily ops discord applications upload-application-attachment`

POST /applications/{application_id}/attachment

Flags: --body

## `coily ops discord channels add-group-dm-user`

PUT /channels/{channel_id}/recipients/{user_id}

Flags: --body

## `coily ops discord channels add-my-message-reaction`

PUT /channels/{channel_id}/messages/{message_id}/reactions/{emoji_name}/@me

## `coily ops discord channels add-thread-member`

PUT /channels/{channel_id}/thread-members/{user_id}

## `coily ops discord channels bulk-delete-messages`

POST /channels/{channel_id}/messages/bulk-delete

Flags: --body

## `coily ops discord channels create-channel-invite`

POST /channels/{channel_id}/invites

Flags: --body

## `coily ops discord channels create-message`

POST /channels/{channel_id}/messages

Flags: --body

## `coily ops discord channels create-pin`

PUT /channels/{channel_id}/messages/pins/{message_id}

## `coily ops discord channels create-thread`

POST /channels/{channel_id}/threads

Flags: --body

## `coily ops discord channels create-thread-from-message`

POST /channels/{channel_id}/messages/{message_id}/threads

Flags: --body

## `coily ops discord channels create-webhook`

POST /channels/{channel_id}/webhooks

Flags: --body

## `coily ops discord channels crosspost-message`

POST /channels/{channel_id}/messages/{message_id}/crosspost

## `coily ops discord channels delete-all-message-reactions`

DELETE /channels/{channel_id}/messages/{message_id}/reactions

## `coily ops discord channels delete-all-message-reactions-by-emoji`

DELETE /channels/{channel_id}/messages/{message_id}/reactions/{emoji_name}

## `coily ops discord channels delete-channel`

DELETE /channels/{channel_id}

## `coily ops discord channels delete-channel-permission-overwrite`

DELETE /channels/{channel_id}/permissions/{overwrite_id}

## `coily ops discord channels delete-group-dm-user`

DELETE /channels/{channel_id}/recipients/{user_id}

## `coily ops discord channels delete-message`

DELETE /channels/{channel_id}/messages/{message_id}

## `coily ops discord channels delete-my-message-reaction`

DELETE /channels/{channel_id}/messages/{message_id}/reactions/{emoji_name}/@me

## `coily ops discord channels delete-pin`

DELETE /channels/{channel_id}/messages/pins/{message_id}

## `coily ops discord channels delete-thread-member`

DELETE /channels/{channel_id}/thread-members/{user_id}

## `coily ops discord channels delete-user-message-reaction`

DELETE /channels/{channel_id}/messages/{message_id}/reactions/{emoji_name}/{user_id}

## `coily ops discord channels deprecated-create-pin`

PUT /channels/{channel_id}/pins/{message_id}

## `coily ops discord channels deprecated-delete-pin`

DELETE /channels/{channel_id}/pins/{message_id}

## `coily ops discord channels deprecated-list-pins`

GET /channels/{channel_id}/pins

## `coily ops discord channels follow-channel`

POST /channels/{channel_id}/followers

Flags: --body

## `coily ops discord channels get-answer-voters`

GET /channels/{channel_id}/polls/{message_id}/answers/{answer_id}

Flags: --after, --limit

## `coily ops discord channels get-channel`

GET /channels/{channel_id}

## `coily ops discord channels get-message`

GET /channels/{channel_id}/messages/{message_id}

## `coily ops discord channels get-thread-member`

GET /channels/{channel_id}/thread-members/{user_id}

Flags: --with_member

## `coily ops discord channels join-thread`

PUT /channels/{channel_id}/thread-members/@me

## `coily ops discord channels leave-thread`

DELETE /channels/{channel_id}/thread-members/@me

## `coily ops discord channels list-channel-invites`

GET /channels/{channel_id}/invites

## `coily ops discord channels list-channel-webhooks`

GET /channels/{channel_id}/webhooks

## `coily ops discord channels list-message-reactions-by-emoji`

GET /channels/{channel_id}/messages/{message_id}/reactions/{emoji_name}

Flags: --after, --limit, --type

## `coily ops discord channels list-messages`

GET /channels/{channel_id}/messages

Flags: --after, --around, --before, --limit

## `coily ops discord channels list-my-private-archived-threads`

GET /channels/{channel_id}/users/@me/threads/archived/private

Flags: --before, --limit

## `coily ops discord channels list-pins`

GET /channels/{channel_id}/messages/pins

Flags: --before, --limit

## `coily ops discord channels list-private-archived-threads`

GET /channels/{channel_id}/threads/archived/private

Flags: --before, --limit

## `coily ops discord channels list-public-archived-threads`

GET /channels/{channel_id}/threads/archived/public

Flags: --before, --limit

## `coily ops discord channels list-thread-members`

GET /channels/{channel_id}/thread-members

Flags: --after, --limit, --with_member

## `coily ops discord channels poll-expire`

POST /channels/{channel_id}/polls/{message_id}/expire

## `coily ops discord channels send-soundboard-sound`

POST /channels/{channel_id}/send-soundboard-sound

Flags: --body

## `coily ops discord channels set-channel-permission-overwrite`

PUT /channels/{channel_id}/permissions/{overwrite_id}

Flags: --body

## `coily ops discord channels thread-search`

GET /channels/{channel_id}/threads/search

Flags: --archived, --limit, --max_id, --min_id, --name, --offset, --slop, --sort_by, --sort_order, --tag, --tag_setting

## `coily ops discord channels trigger-typing-indicator`

POST /channels/{channel_id}/typing

## `coily ops discord channels update-channel`

PATCH /channels/{channel_id}

Flags: --body

## `coily ops discord channels update-message`

PATCH /channels/{channel_id}/messages/{message_id}

Flags: --body

## `coily ops discord channels update-voice-channel-status`

Set a voice channel's status.

Flags: --body

## `coily ops discord gateway get-bot-gateway`

GET /gateway/bot

## `coily ops discord gateway get-gateway`

GET /gateway

## `coily ops discord guilds action-guild-join-request`

Approve or reject guild join request

Flags: --body

## `coily ops discord guilds add-guild-member`

PUT /guilds/{guild_id}/members/{user_id}

Flags: --body

## `coily ops discord guilds add-guild-member-role`

PUT /guilds/{guild_id}/members/{user_id}/roles/{role_id}

## `coily ops discord guilds ban-user-from-guild`

PUT /guilds/{guild_id}/bans/{user_id}

Flags: --body

## `coily ops discord guilds bulk-ban-users-from-guild`

POST /guilds/{guild_id}/bulk-ban

Flags: --body

## `coily ops discord guilds bulk-update-guild-channels`

PATCH /guilds/{guild_id}/channels

Flags: --body

## `coily ops discord guilds bulk-update-guild-roles`

PATCH /guilds/{guild_id}/roles

Flags: --body

## `coily ops discord guilds create-auto-moderation-rule`

POST /guilds/{guild_id}/auto-moderation/rules

Flags: --body

## `coily ops discord guilds create-guild-channel`

POST /guilds/{guild_id}/channels

Flags: --body

## `coily ops discord guilds create-guild-emoji`

POST /guilds/{guild_id}/emojis

Flags: --body

## `coily ops discord guilds create-guild-role`

POST /guilds/{guild_id}/roles

Flags: --body

## `coily ops discord guilds create-guild-scheduled-event`

POST /guilds/{guild_id}/scheduled-events

Flags: --body

## `coily ops discord guilds create-guild-soundboard-sound`

POST /guilds/{guild_id}/soundboard-sounds

Flags: --body

## `coily ops discord guilds create-guild-sticker`

POST /guilds/{guild_id}/stickers

Flags: --body

## `coily ops discord guilds create-guild-template`

POST /guilds/{guild_id}/templates

Flags: --body

## `coily ops discord guilds delete-auto-moderation-rule`

DELETE /guilds/{guild_id}/auto-moderation/rules/{rule_id}

## `coily ops discord guilds delete-guild-emoji`

DELETE /guilds/{guild_id}/emojis/{emoji_id}

## `coily ops discord guilds delete-guild-integration`

DELETE /guilds/{guild_id}/integrations/{integration_id}

## `coily ops discord guilds delete-guild-member`

DELETE /guilds/{guild_id}/members/{user_id}

## `coily ops discord guilds delete-guild-member-role`

DELETE /guilds/{guild_id}/members/{user_id}/roles/{role_id}

## `coily ops discord guilds delete-guild-role`

DELETE /guilds/{guild_id}/roles/{role_id}

## `coily ops discord guilds delete-guild-scheduled-event`

DELETE /guilds/{guild_id}/scheduled-events/{guild_scheduled_event_id}

## `coily ops discord guilds delete-guild-soundboard-sound`

DELETE /guilds/{guild_id}/soundboard-sounds/{sound_id}

## `coily ops discord guilds delete-guild-sticker`

DELETE /guilds/{guild_id}/stickers/{sticker_id}

## `coily ops discord guilds delete-guild-template`

DELETE /guilds/{guild_id}/templates/{code}

## `coily ops discord guilds get-active-guild-threads`

GET /guilds/{guild_id}/threads/active

## `coily ops discord guilds get-auto-moderation-rule`

GET /guilds/{guild_id}/auto-moderation/rules/{rule_id}

## `coily ops discord guilds get-guild`

GET /guilds/{guild_id}

Flags: --with_counts

## `coily ops discord guilds get-guild-ban`

GET /guilds/{guild_id}/bans/{user_id}

## `coily ops discord guilds get-guild-emoji`

GET /guilds/{guild_id}/emojis/{emoji_id}

## `coily ops discord guilds get-guild-join-requests`

List join requests for guild, optionally filtered by application status

Flags: --after, --before, --limit, --status

## `coily ops discord guilds get-guild-member`

GET /guilds/{guild_id}/members/{user_id}

## `coily ops discord guilds get-guild-new-member-welcome`

GET /guilds/{guild_id}/new-member-welcome

## `coily ops discord guilds get-guild-preview`

GET /guilds/{guild_id}/preview

## `coily ops discord guilds get-guild-role`

GET /guilds/{guild_id}/roles/{role_id}

## `coily ops discord guilds get-guild-scheduled-event`

GET /guilds/{guild_id}/scheduled-events/{guild_scheduled_event_id}

Flags: --with_user_count

## `coily ops discord guilds get-guild-soundboard-sound`

GET /guilds/{guild_id}/soundboard-sounds/{sound_id}

## `coily ops discord guilds get-guild-sticker`

GET /guilds/{guild_id}/stickers/{sticker_id}

## `coily ops discord guilds get-guild-template`

GET /guilds/templates/{code}

## `coily ops discord guilds get-guild-vanity-url`

GET /guilds/{guild_id}/vanity-url

## `coily ops discord guilds get-guild-webhooks`

GET /guilds/{guild_id}/webhooks

## `coily ops discord guilds get-guild-welcome-screen`

GET /guilds/{guild_id}/welcome-screen

## `coily ops discord guilds get-guild-widget`

GET /guilds/{guild_id}/widget.json

## `coily ops discord guilds get-guild-widget-png`

GET /guilds/{guild_id}/widget.png

Flags: --style

## `coily ops discord guilds get-guild-widget-settings`

GET /guilds/{guild_id}/widget

## `coily ops discord guilds get-guilds-onboarding`

GET /guilds/{guild_id}/onboarding

## `coily ops discord guilds get-self-voice-state`

GET /guilds/{guild_id}/voice-states/@me

## `coily ops discord guilds get-voice-state`

GET /guilds/{guild_id}/voice-states/{user_id}

## `coily ops discord guilds guild-role-member-counts`

GET /guilds/{guild_id}/roles/member-counts

## `coily ops discord guilds guild-search`

GET /guilds/{guild_id}/messages/search

Flags: --attachment_extension, --attachment_filename, --author_id, --author_type, --channel_id, --content, --embed_provider, --embed_type, --has, --include_nsfw, --limit, --link_hostname, --max_id, --mention_everyone, --mentions, --mentions_role_id, --min_id, --offset, --pinned, --replied_to_message_id, --replied_to_user_id, --slop, --sort_by, --sort_order

## `coily ops discord guilds list-auto-moderation-rules`

GET /guilds/{guild_id}/auto-moderation/rules

## `coily ops discord guilds list-guild-audit-log-entries`

GET /guilds/{guild_id}/audit-logs

Flags: --action_type, --after, --before, --limit, --target_id, --user_id

## `coily ops discord guilds list-guild-bans`

GET /guilds/{guild_id}/bans

Flags: --after, --before, --limit

## `coily ops discord guilds list-guild-channels`

GET /guilds/{guild_id}/channels

## `coily ops discord guilds list-guild-emojis`

GET /guilds/{guild_id}/emojis

## `coily ops discord guilds list-guild-integrations`

GET /guilds/{guild_id}/integrations

## `coily ops discord guilds list-guild-invites`

GET /guilds/{guild_id}/invites

## `coily ops discord guilds list-guild-members`

GET /guilds/{guild_id}/members

Flags: --after, --limit

## `coily ops discord guilds list-guild-roles`

GET /guilds/{guild_id}/roles

## `coily ops discord guilds list-guild-scheduled-event-users`

GET /guilds/{guild_id}/scheduled-events/{guild_scheduled_event_id}/users

Flags: --after, --before, --limit, --with_member

## `coily ops discord guilds list-guild-scheduled-events`

GET /guilds/{guild_id}/scheduled-events

Flags: --with_user_count

## `coily ops discord guilds list-guild-soundboard-sounds`

GET /guilds/{guild_id}/soundboard-sounds

## `coily ops discord guilds list-guild-stickers`

GET /guilds/{guild_id}/stickers

## `coily ops discord guilds list-guild-templates`

GET /guilds/{guild_id}/templates

## `coily ops discord guilds list-guild-voice-regions`

GET /guilds/{guild_id}/regions

## `coily ops discord guilds preview-prune-guild`

GET /guilds/{guild_id}/prune

Flags: --days, --include_roles

## `coily ops discord guilds prune-guild`

POST /guilds/{guild_id}/prune

Flags: --body

## `coily ops discord guilds put-guilds-onboarding`

PUT /guilds/{guild_id}/onboarding

Flags: --body

## `coily ops discord guilds search-guild-members`

GET /guilds/{guild_id}/members/search

Flags: --limit, --query

## `coily ops discord guilds sync-guild-template`

PUT /guilds/{guild_id}/templates/{code}

## `coily ops discord guilds unban-user-from-guild`

DELETE /guilds/{guild_id}/bans/{user_id}

Flags: --body

## `coily ops discord guilds update-auto-moderation-rule`

PATCH /guilds/{guild_id}/auto-moderation/rules/{rule_id}

Flags: --body

## `coily ops discord guilds update-guild`

PATCH /guilds/{guild_id}

Flags: --body

## `coily ops discord guilds update-guild-emoji`

PATCH /guilds/{guild_id}/emojis/{emoji_id}

Flags: --body

## `coily ops discord guilds update-guild-member`

PATCH /guilds/{guild_id}/members/{user_id}

Flags: --body

## `coily ops discord guilds update-guild-role`

PATCH /guilds/{guild_id}/roles/{role_id}

Flags: --body

## `coily ops discord guilds update-guild-scheduled-event`

PATCH /guilds/{guild_id}/scheduled-events/{guild_scheduled_event_id}

Flags: --body

## `coily ops discord guilds update-guild-soundboard-sound`

PATCH /guilds/{guild_id}/soundboard-sounds/{sound_id}

Flags: --body

## `coily ops discord guilds update-guild-sticker`

PATCH /guilds/{guild_id}/stickers/{sticker_id}

Flags: --body

## `coily ops discord guilds update-guild-template`

PATCH /guilds/{guild_id}/templates/{code}

Flags: --body

## `coily ops discord guilds update-guild-welcome-screen`

PATCH /guilds/{guild_id}/welcome-screen

Flags: --body

## `coily ops discord guilds update-guild-widget-settings`

PATCH /guilds/{guild_id}/widget

Flags: --body

## `coily ops discord guilds update-my-guild-member`

PATCH /guilds/{guild_id}/members/@me

Flags: --body

## `coily ops discord guilds update-self-voice-state`

PATCH /guilds/{guild_id}/voice-states/@me

Flags: --body

## `coily ops discord guilds update-voice-state`

PATCH /guilds/{guild_id}/voice-states/{user_id}

Flags: --body

## `coily ops discord interactions create-interaction-response`

POST /interactions/{interaction_id}/{interaction_token}/callback

Flags: --body, --with_response

## `coily ops discord invites get-invite-target-users`

Get the target users for an invite.

## `coily ops discord invites get-invite-target-users-job-status`

Get the target users job status for an invite.

## `coily ops discord invites invite-resolve`

GET /invites/{code}

Flags: --guild_scheduled_event_id, --with_counts

## `coily ops discord invites invite-revoke`

DELETE /invites/{code}

## `coily ops discord invites update-invite-target-users`

Update the target users for an existing invite.

Flags: --body

## `coily ops discord lobbies add-lobby-member`

PUT /lobbies/{lobby_id}/members/{user_id}

Flags: --body

## `coily ops discord lobbies bulk-update-lobby-members`

POST /lobbies/{lobby_id}/members/bulk

Flags: --body

## `coily ops discord lobbies create-linked-lobby-guild-invite-for-self`

POST /lobbies/{lobby_id}/members/@me/invites

## `coily ops discord lobbies create-linked-lobby-guild-invite-for-user`

POST /lobbies/{lobby_id}/members/{user_id}/invites

## `coily ops discord lobbies create-lobby`

POST /lobbies

Flags: --body

## `coily ops discord lobbies create-lobby-message`

POST /lobbies/{lobby_id}/messages

Flags: --body

## `coily ops discord lobbies create-or-join-lobby`

PUT /lobbies

Flags: --body

## `coily ops discord lobbies delete-lobby-member`

DELETE /lobbies/{lobby_id}/members/{user_id}

## `coily ops discord lobbies edit-lobby`

PATCH /lobbies/{lobby_id}

Flags: --body

## `coily ops discord lobbies edit-lobby-channel-link`

PATCH /lobbies/{lobby_id}/channel-linking

Flags: --body

## `coily ops discord lobbies get-lobby`

GET /lobbies/{lobby_id}

## `coily ops discord lobbies get-lobby-messages`

GET /lobbies/{lobby_id}/messages

Flags: --limit

## `coily ops discord lobbies leave-lobby`

DELETE /lobbies/{lobby_id}/members/@me

## `coily ops discord lobbies update-lobby-message-external-moderation-metadata`

Update the external moderation metadata for a lobby message.

Flags: --body

## `coily ops discord oauth2 get-my-oauth2-application`

GET /oauth2/applications/@me

## `coily ops discord oauth2 get-my-oauth2-authorization`

GET /oauth2/@me

## `coily ops discord oauth2 get-openid-connect-userinfo`

GET /oauth2/userinfo

## `coily ops discord oauth2 get-public-keys`

GET /oauth2/keys

## `coily ops discord partner-sdk bot-partner-sdk-token`

POST /partner-sdk/token/bot

Flags: --body

## `coily ops discord partner-sdk bot-partner-sdk-unmerge-provisional-account`

POST /partner-sdk/provisional-accounts/unmerge/bot

Flags: --body

## `coily ops discord partner-sdk partner-sdk-token`

POST /partner-sdk/token

Flags: --body

## `coily ops discord partner-sdk partner-sdk-unmerge-provisional-account`

POST /partner-sdk/provisional-accounts/unmerge

Flags: --body

## `coily ops discord partner-sdk update-user-message-external-moderation-metadata`

Update the external moderation metadata for a user message (DM).

Flags: --body

## `coily ops discord soundboard-default-sounds get-soundboard-default-sounds`

GET /soundboard-default-sounds

## `coily ops discord stage-instances create-stage-instance`

POST /stage-instances

Flags: --body

## `coily ops discord stage-instances delete-stage-instance`

DELETE /stage-instances/{channel_id}

## `coily ops discord stage-instances get-stage-instance`

GET /stage-instances/{channel_id}

## `coily ops discord stage-instances update-stage-instance`

PATCH /stage-instances/{channel_id}

Flags: --body

## `coily ops discord sticker-packs get-sticker-pack`

GET /sticker-packs/{pack_id}

## `coily ops discord sticker-packs list-sticker-packs`

GET /sticker-packs

## `coily ops discord stickers get-sticker`

GET /stickers/{sticker_id}

## `coily ops discord users create-dm`

POST /users/@me/channels

Flags: --body

## `coily ops discord users delete-application-user-role-connection`

DELETE /users/@me/applications/{application_id}/role-connection

## `coily ops discord users get-application-user-role-connection`

GET /users/@me/applications/{application_id}/role-connection

## `coily ops discord users get-current-user-application-entitlements`

GET /users/@me/applications/{application_id}/entitlements

Flags: --exclude_consumed, --sku_ids

## `coily ops discord users get-my-guild-member`

GET /users/@me/guilds/{guild_id}/member

## `coily ops discord users get-my-user`

GET /users/@me

## `coily ops discord users get-user`

GET /users/{user_id}

## `coily ops discord users leave-guild`

DELETE /users/@me/guilds/{guild_id}

## `coily ops discord users list-my-connections`

GET /users/@me/connections

## `coily ops discord users list-my-guilds`

GET /users/@me/guilds

Flags: --after, --before, --limit, --with_counts

## `coily ops discord users update-application-user-role-connection`

PUT /users/@me/applications/{application_id}/role-connection

Flags: --body

## `coily ops discord users update-my-user`

PATCH /users/@me

Flags: --body

## `coily ops discord voice list-voice-regions`

GET /voice/regions

## `coily ops discord webhooks delete-original-webhook-message`

DELETE /webhooks/{webhook_id}/{webhook_token}/messages/@original

Flags: --thread_id

## `coily ops discord webhooks delete-webhook`

DELETE /webhooks/{webhook_id}

## `coily ops discord webhooks delete-webhook-by-token`

DELETE /webhooks/{webhook_id}/{webhook_token}

## `coily ops discord webhooks delete-webhook-message`

DELETE /webhooks/{webhook_id}/{webhook_token}/messages/{message_id}

Flags: --thread_id

## `coily ops discord webhooks execute-github-compatible-webhook`

POST /webhooks/{webhook_id}/{webhook_token}/github

Flags: --body, --thread_id, --wait

## `coily ops discord webhooks execute-slack-compatible-webhook`

POST /webhooks/{webhook_id}/{webhook_token}/slack

Flags: --body, --thread_id, --wait

## `coily ops discord webhooks execute-webhook`

POST /webhooks/{webhook_id}/{webhook_token}

Flags: --body, --thread_id, --wait, --with_components

## `coily ops discord webhooks get-original-webhook-message`

GET /webhooks/{webhook_id}/{webhook_token}/messages/@original

Flags: --thread_id

## `coily ops discord webhooks get-webhook`

GET /webhooks/{webhook_id}

## `coily ops discord webhooks get-webhook-by-token`

GET /webhooks/{webhook_id}/{webhook_token}

## `coily ops discord webhooks get-webhook-message`

GET /webhooks/{webhook_id}/{webhook_token}/messages/{message_id}

Flags: --thread_id

## `coily ops discord webhooks update-original-webhook-message`

PATCH /webhooks/{webhook_id}/{webhook_token}/messages/@original

Flags: --body, --thread_id, --with_components

## `coily ops discord webhooks update-webhook`

PATCH /webhooks/{webhook_id}

Flags: --body

## `coily ops discord webhooks update-webhook-by-token`

PATCH /webhooks/{webhook_id}/{webhook_token}

Flags: --body

## `coily ops discord webhooks update-webhook-message`

PATCH /webhooks/{webhook_id}/{webhook_token}/messages/{message_id}

Flags: --body, --thread_id, --with_components

## `coily ops flyctl`

Pass-through to flyctl with argv validation + audit log.

## `coily ops forgejo admin auth list`

List forgejo auth sources.

## `coily ops forgejo admin user create`

Create a forgejo user with a random password and forced first-login rotation.

Flags: --admin, --email, --username

## `coily ops forgejo admin user list`

List forgejo users.

## `coily ops forgejo doctor check`

Run a forgejo doctor check (readonly; --fix not exposed).

Flags: --run

## `coily ops gh`

Pass-through to gh with argv validation + audit log.

## `coily ops kubectl`

Pass-through to kubectl with argv validation + audit log.

## `coily ops modio mods comments`

GET /games/{game-id}/mods/{mod-id}/comments

## `coily ops modio mods files`

GET /games/{game-id}/mods/{mod-id}/files

## `coily ops modio mods get`

GET /games/{game-id}/mods/{mod-id}

## `coily ops modio mods list`

GET /games/{game-id}/mods

Flags: --limit, --offset

## `coily ops personal-dashboard restart`

Restart the personal-dashboard unit.

## `coily ops personal-dashboard start`

Start the personal-dashboard unit.

## `coily ops personal-dashboard status`

Print systemctl status personal-dashboard.

## `coily ops personal-dashboard stop`

Stop the personal-dashboard unit.

## `coily ops personal-dashboard tail`

Tail personal-dashboard journal logs (journalctl -u personal-dashboard -f).

Flags: --follow, --lines

## `coily ops sentry events bulk-mutate-a-list-of-issues`

Bulk mutate various attributes on issues. The list of issues to modify is given through the `id` query parameter. It is…

Flags: --body, --id, --status

## `coily ops sentry events bulk-remove-a-list-of-issues`

Permanently remove the given issues. The list of issues to modify is given through the `id` query parameter. It is repe…

Flags: --id

## `coily ops sentry events list-a-project-s-issues`

**Deprecated**: This endpoint has been replaced with the [Organization Issues endpoint](/api/events/list-an-organizatio…

Flags: --hashes, --query, --shortIdLookup, --statsPeriod

## `coily ops sentry events list-a-tag-s-values-related-to-an-issue`

Returns details for given tag key related to an issue. When [paginated](/api/pagination) can return at most 1000 values.

## `coily ops sentry events list-an-issue-s-hashes`

This endpoint lists an issue's hashes, which are the generated checksums used to aggregate individual events.

Flags: --full

## `coily ops sentry events remove-an-issue`

Removes an individual issue.

## `coily ops sentry events retrieve-an-event-for-a-project`

Return details on an individual event.

## `coily ops sentry events retrieve-an-issue`

Return details on an individual issue. This returns the basic stats for the issue (title, last seen, first seen), some…

Flags: --collapse

## `coily ops sentry events update-an-issue`

Updates an individual issue's attributes. Only the attributes submitted are modified.

Flags: --body

## `coily ops sentry integration create-or-update-an-external-issue`

Create or update an external issue from an integration platform integration.

Flags: --body

## `coily ops sentry integration delete-an-external-issue`

Delete an external issue.

## `coily ops sentry integration list-an-organization-s-integration-platform-installations`

Return a list of integration platform installations for a given organization.

## `coily ops sentry organizations list-an-organization-s-repositories`

Return a list of version control repositories for a given organization.

## `coily ops sentry projects delete-a-specific-project-s-debug-information-file`

Delete a debug information file for a given project.

Flags: --id

## `coily ops sentry projects disable-spike-protection`

Disables Spike Protection feature for some of the projects within the organization.

Flags: --body

## `coily ops sentry projects enable-spike-protection`

Enables Spike Protection feature for some of the projects within the organization.

Flags: --body

## `coily ops sentry projects list-a-project-s-debug-information-files`

Retrieve a list of debug information files for a given project.

## `coily ops sentry projects list-a-project-s-service-hooks`

Return a list of service hooks bound to a project.

## `coily ops sentry projects list-a-project-s-user-feedback`

Return a list of user feedback items within this project. *This list does not include submissions from the [User Feedba…

## `coily ops sentry projects list-a-project-s-users`

Return a list of users seen within this project.

Flags: --query

## `coily ops sentry projects list-a-tag-s-values`

Return a list of values associated with this key. The `query` parameter can be used to to perform a 'contains' match on…

## `coily ops sentry projects register-a-new-service-hook`

Register a new service hook on a project. Events include: - event.alert: An alert is generated for an event (via rules)…

Flags: --body

## `coily ops sentry projects remove-a-service-hook`

Remove a service hook.

## `coily ops sentry projects retrieve-a-service-hook`

Return a service hook bound to a project.

## `coily ops sentry projects retrieve-event-counts-for-a-project`

Caution This endpoint may change in the future without notice.

Flags: --resolution, --since, --stat, --until

## `coily ops sentry projects submit-user-feedback`

*This endpoint is DEPRECATED. We document it here for older SDKs and users who are still migrating to the [User Feedbac…

Flags: --body

## `coily ops sentry projects update-a-service-hook`

Update a service hook.

Flags: --body

## `coily ops sentry projects upload-a-new-file`

Upload a new debug information file for the given release. Unlike other API requests, files must be uploaded using the…

Flags: --body

## `coily ops sentry releases create-a-new-release-for-an-organization`

Create a new release for the given organization. Releases are used by Sentry to improve its error reporting abilities b…

Flags: --body

## `coily ops sentry releases delete-a-project-release-s-file`

Delete a file for a given release.

## `coily ops sentry releases delete-an-organization-release-s-file`

Delete a file for a given release.

## `coily ops sentry releases list-a-project-release-s-commits`

List a project release's commits.

## `coily ops sentry releases list-a-project-s-release-files`

Return a list of files for a given release.

## `coily ops sentry releases list-an-organization-release-s-commits`

List an organization release's commits.

## `coily ops sentry releases list-an-organization-s-release-files`

Return a list of files for a given release.

## `coily ops sentry releases list-an-organization-s-releases`

Return a list of releases for a given organization.

Flags: --query

## `coily ops sentry releases retrieve-a-project-release-s-file`

Retrieve a file for a given release.

Flags: --download

## `coily ops sentry releases retrieve-an-organization-release-s-file`

Retrieve a file for a given release.

Flags: --download

## `coily ops sentry releases retrieve-files-changed-in-a-release-s-commits`

Retrieve files changed in a release's commits

## `coily ops sentry releases update-a-project-release-file`

Update a project release file.

Flags: --body

## `coily ops sentry releases update-an-organization-release-file`

Update an organization release file.

Flags: --body

## `coily ops sentry releases upload-a-new-organization-release-file`

Upload a new file for the given release. Unlike other API requests, files must be uploaded using the traditional multip…

Flags: --body

## `coily ops sentry releases upload-a-new-project-release-file`

Upload a new file for the given release. Unlike other API requests, files must be uploaded using the traditional multip…

Flags: --body

## `coily ops trello actions delete-actions-id`

Delete an Action

## `coily ops trello actions delete-actions-idaction-reactions-id`

Delete Action's Reaction

## `coily ops trello actions get-actions-id`

Get an Action

Flags: --display, --entities, --fields, --member, --memberCreator, --memberCreator_fields, --member_fields

## `coily ops trello actions get-actions-id-board`

Get the Board for an Action

Flags: --fields

## `coily ops trello actions get-actions-id-card`

Get the Card for an Action

Flags: --fields

## `coily ops trello actions get-actions-id-field`

Get a specific field on an Action

## `coily ops trello actions get-actions-id-list`

Get the List for an Action

Flags: --fields

## `coily ops trello actions get-actions-id-member`

Get the Member of an Action

Flags: --fields

## `coily ops trello actions get-actions-id-membercreator`

Get the Member Creator of an Action

Flags: --fields

## `coily ops trello actions get-actions-id-organization`

Get the Organization of an Action

Flags: --fields

## `coily ops trello actions get-actions-idaction-reactions`

Get Action's Reactions

Flags: --emoji, --member

## `coily ops trello actions get-actions-idaction-reactions-id`

Get Action's Reaction

Flags: --emoji, --member

## `coily ops trello actions get-actions-idaction-reactionsummary`

List Action's summary of Reactions

## `coily ops trello actions post-actions-idaction-reactions`

Create Reaction for Action

Flags: --body

## `coily ops trello actions put-actions-id`

Update an Action

Flags: --text

## `coily ops trello actions put-actions-id-text`

Update a Comment Action

Flags: --value

## `coily ops trello applications applications-key-compliance`

Get Application's compliance data

## `coily ops trello batch get-batch`

Batch Requests

Flags: --urls

## `coily ops trello boards boards-id-checklists`

Get Checklists on a Board

## `coily ops trello boards boardsidmembersidmember`

Remove Member from Board

## `coily ops trello boards delete-boards-id`

Delete a Board

## `coily ops trello boards get-board-id-plugins`

Get Power-Ups on a Board

Flags: --filter

## `coily ops trello boards get-boards-id`

Get a Board

Flags: --actions, --boardStars, --card_pluginData, --cards, --checklists, --customFields, --fields, --labels, --lists, --members, --memberships, --myPrefs, --organization, --organization_pluginData, --pluginData, --tags

## `coily ops trello boards get-boards-id-actions`

Get Actions of a Board

Flags: --before, --fields, --filter, --format, --idModels, --limit, --member, --memberCreator, --memberCreator_fields, --member_fields, --page, --reactions, --since

## `coily ops trello boards get-boards-id-boardplugins`

Get Enabled Power-Ups on Board

## `coily ops trello boards get-boards-id-boardstars`

Get boardStars on a Board

Flags: --filter

## `coily ops trello boards get-boards-id-cards`

Get Cards on a Board

## `coily ops trello boards get-boards-id-cards-filter`

Get filtered Cards on a Board

## `coily ops trello boards get-boards-id-customfields`

Get Custom Fields for Board

## `coily ops trello boards get-boards-id-field`

Get a field on a Board

## `coily ops trello boards get-boards-id-labels`

Get Labels on a Board

Flags: --fields, --limit

## `coily ops trello boards get-boards-id-lists`

Get Lists on a Board

Flags: --card_fields, --cards, --fields, --filter

## `coily ops trello boards get-boards-id-lists-filter`

Get filtered Lists on a Board

## `coily ops trello boards get-boards-id-members`

Get the Members of a Board

## `coily ops trello boards get-boards-id-memberships`

Get Memberships of a Board

Flags: --activity, --filter, --member, --member_fields, --orgMemberType

## `coily ops trello boards post-boards`

Create a Board

Flags: --defaultLabels, --defaultLists, --desc, --idBoardSource, --idOrganization, --keepFromSource, --name, --powerUps, --prefs_background, --prefs_cardAging, --prefs_cardCovers, --prefs_comments, --prefs_invitations, --prefs_permissionLevel, --prefs_selfJoin, --prefs_voting

## `coily ops trello boards post-boards-id-calendarkey-generate`

Create a calendarKey for a Board

## `coily ops trello boards post-boards-id-emailkey-generate`

Create a emailKey for a Board

## `coily ops trello boards post-boards-id-idtags`

Create a Tag for a Board

Flags: --value

## `coily ops trello boards post-boards-id-labels`

Create a Label on a Board

Flags: --color, --name

## `coily ops trello boards post-boards-id-lists`

Create a List on a Board

Flags: --name, --pos

## `coily ops trello boards post-boards-id-markedasviewed`

Mark Board as viewed

## `coily ops trello boards put-boards-id`

Update a Board

Flags: --closed, --desc, --idOrganization, --name, --prefs/background, --prefs/calendarFeedEnabled, --prefs/cardAging, --prefs/cardCovers, --prefs/comments, --prefs/hideVotes, --prefs/invitations, --prefs/permissionLevel, --prefs/selfJoin, --prefs/voting, --subscribed

## `coily ops trello boards put-boards-id-members`

Invite Member to Board via email

Flags: --body, --email, --type

## `coily ops trello boards put-boards-id-members-idmember`

Add a Member to a Board

Flags: --allowBillableGuest, --type

## `coily ops trello boards put-boards-id-memberships-idmembership`

Update Membership of Member on a Board

Flags: --member_fields, --type

## `coily ops trello boards put-boards-id-my-prefs-showsidebar`

Update showSidebar Pref on a Board

Flags: --value

## `coily ops trello boards put-boards-id-my-prefs-showsidebaractivity`

Update showSidebarActivity Pref on a Board

Flags: --value

## `coily ops trello boards put-boards-id-my-prefs-showsidebarboardactions`

Update showSidebarBoardActions Pref on a Board

Flags: --value

## `coily ops trello boards put-boards-id-my-prefs-showsidebarmembers`

Update showSidebarMembers Pref on a Board

Flags: --value

## `coily ops trello boards put-boards-id-myprefs-emailposition`

Update emailPosition Pref on a Board

Flags: --value

## `coily ops trello boards put-boards-id-myprefs-idemaillist`

Update idEmailList Pref on a Board

Flags: --value

## `coily ops trello cards cardsidmembersvoted-1`

Add Member vote to Card

Flags: --value

## `coily ops trello cards delete-cards-id`

Delete a Card

## `coily ops trello cards delete-cards-id-actions-id-comments`

Delete a comment on a Card

## `coily ops trello cards delete-cards-id-checkitem-idcheckitem`

Delete checkItem on a Card

## `coily ops trello cards delete-cards-id-checklists-idchecklist`

Delete a Checklist on a Card

## `coily ops trello cards delete-cards-id-idlabels-idlabel`

Remove a Label from a Card

## `coily ops trello cards delete-cards-id-membersvoted-idmember`

Remove a Member's Vote on a Card

## `coily ops trello cards delete-cards-id-stickers-idsticker`

Delete a Sticker on a Card

## `coily ops trello cards delete-id-idmembers-idmember`

Remove a Member from a Card

## `coily ops trello cards deleted-cards-id-attachments-idattachment`

Delete an Attachment on a Card

## `coily ops trello cards get-cards-id`

Get a Card

Flags: --actions, --attachment_fields, --attachments, --board, --board_fields, --checkItemStates, --checklist_fields, --checklists, --customFieldItems, --fields, --list, --memberVoted_fields, --member_fields, --members, --membersVoted, --pluginData, --sticker_fields, --stickers

## `coily ops trello cards get-cards-id-actions`

Get Actions on a Card

Flags: --filter, --page

## `coily ops trello cards get-cards-id-attachments`

Get Attachments on a Card

Flags: --fields, --filter

## `coily ops trello cards get-cards-id-attachments-idattachment`

Get an Attachment on a Card

Flags: --fields

## `coily ops trello cards get-cards-id-board`

Get the Board the Card is on

Flags: --fields

## `coily ops trello cards get-cards-id-checkitem-idcheckitem`

Get checkItem on a Card

Flags: --fields

## `coily ops trello cards get-cards-id-checkitemstates`

Get checkItems on a Card

Flags: --fields

## `coily ops trello cards get-cards-id-checklists`

Get Checklists on a Card

Flags: --checkItem_fields, --checkItems, --fields, --filter

## `coily ops trello cards get-cards-id-customfielditems`

Get Custom Field Items for a Card

## `coily ops trello cards get-cards-id-field`

Get a field on a Card

## `coily ops trello cards get-cards-id-list`

Get the List of a Card

Flags: --fields

## `coily ops trello cards get-cards-id-members`

Get the Members of a Card

Flags: --fields

## `coily ops trello cards get-cards-id-membersvoted`

Get Members who have voted on a Card

Flags: --fields

## `coily ops trello cards get-cards-id-plugindata`

Get pluginData on a Card

## `coily ops trello cards get-cards-id-stickers`

Get Stickers on a Card

Flags: --fields

## `coily ops trello cards get-cards-id-stickers-idsticker`

Get a Sticker on a Card

Flags: --fields

## `coily ops trello cards post-cards`

Create a new Card

Flags: --address, --cardRole, --coordinates, --desc, --due, --dueComplete, --fileSource, --idCardSource, --idLabels, --idList, --idMembers, --keepFromSource, --locationName, --mimeType, --name, --pos, --start, --urlSource

## `coily ops trello cards post-cards-id-actions-comments`

Add a new comment to a Card

Flags: --text

## `coily ops trello cards post-cards-id-attachments`

Create Attachment On Card

Flags: --file, --mimeType, --name, --setCover, --url

## `coily ops trello cards post-cards-id-checklists`

Create Checklist on a Card

Flags: --idChecklistSource, --name, --pos

## `coily ops trello cards post-cards-id-idlabels`

Add a Label to a Card

Flags: --value

## `coily ops trello cards post-cards-id-idmembers`

Add a Member to a Card

Flags: --value

## `coily ops trello cards post-cards-id-labels`

Create a new Label on a Card

Flags: --color, --name

## `coily ops trello cards post-cards-id-markassociatednotificationsread`

Mark a Card's Notifications as read

## `coily ops trello cards post-cards-id-stickers`

Add a Sticker to a Card

Flags: --image, --left, --rotate, --top, --zIndex

## `coily ops trello cards put-cards-id`

Update a Card

Flags: --address, --closed, --coordinates, --cover, --desc, --due, --dueComplete, --idAttachmentCover, --idBoard, --idLabels, --idList, --idMembers, --locationName, --name, --pos, --start, --subscribed

## `coily ops trello cards put-cards-id-actions-idaction-comments`

Update Comment Action on a Card

Flags: --text

## `coily ops trello cards put-cards-id-checkitem-idcheckitem`

Update a checkItem on a Card

Flags: --due, --dueReminder, --idChecklist, --idMember, --name, --pos, --state

## `coily ops trello cards put-cards-id-stickers-idsticker`

Update a Sticker on a Card

Flags: --left, --rotate, --top, --zIndex

## `coily ops trello cards put-cards-idcard-checklist-idchecklist-checkitem-idcheckitem`

Update Checkitem on Checklist on Card

Flags: --pos

## `coily ops trello cards put-cards-idcard-customfield-idcustomfield-item`

Update Custom Field item on Card

Flags: --body

## `coily ops trello cards put-cards-idcard-customfields`

Update Multiple Custom Field items on Card

Flags: --body

## `coily ops trello checklists delete-checklists-id`

Delete a Checklist

## `coily ops trello checklists delete-checklists-id-checkitems-idcheckitem`

Delete Checkitem from Checklist

## `coily ops trello checklists get-checklists-id`

Get a Checklist

Flags: --cards, --checkItem_fields, --checkItems, --fields

## `coily ops trello checklists get-checklists-id-board`

Get the Board the Checklist is on

Flags: --fields

## `coily ops trello checklists get-checklists-id-cards`

Get the Card a Checklist is on

## `coily ops trello checklists get-checklists-id-checkitems`

Get Checkitems on a Checklist

Flags: --fields, --filter

## `coily ops trello checklists get-checklists-id-checkitems-idcheckitem`

Get a Checkitem on a Checklist

Flags: --fields

## `coily ops trello checklists get-checklists-id-field`

Get field on a Checklist

## `coily ops trello checklists post-checklists`

Create a Checklist

Flags: --idCard, --idChecklistSource, --name, --pos

## `coily ops trello checklists post-checklists-id-checkitems`

Create Checkitem on Checklist

Flags: --checked, --due, --dueReminder, --idMember, --name, --pos

## `coily ops trello checklists put-checklists-id-field`

Update field on a Checklist

Flags: --value

## `coily ops trello checklists put-checlists-id`

Update a Checklist

Flags: --name, --pos

## `coily ops trello custom-fields delete-customfields-id`

Delete a Custom Field definition

## `coily ops trello custom-fields delete-customfields-options-idcustomfieldoption`

Delete Option of Custom Field dropdown

## `coily ops trello custom-fields get-customfields-id`

Get a Custom Field

## `coily ops trello custom-fields get-customfields-id-options`

Add Option to Custom Field dropdown

## `coily ops trello custom-fields get-customfields-options-idcustomfieldoption`

Get Option of Custom Field dropdown

## `coily ops trello custom-fields post-customfields`

Create a new Custom Field on a Board

Flags: --body

## `coily ops trello custom-fields post-customfields-id-options`

Get Options of Custom Field drop down

## `coily ops trello custom-fields put-customfields-id`

Update a Custom Field definition

Flags: --body

## `coily ops trello emoji emoji`

List available Emoji

Flags: --locale, --spritesheets

## `coily ops trello enterprises delete-enterprises-id-organizations-idorg`

Delete an Organization from an Enterprise.

## `coily ops trello enterprises enterprises-id-members-id-member-deactivated`

Deactivate a Member of an Enterprise.

Flags: --board_fields, --fields, --organization_fields, --value

## `coily ops trello enterprises enterprises-id-organizations-idmember`

Remove a Member as admin from Enterprise.

## `coily ops trello enterprises get-enterprises-id`

Get an Enterprise

Flags: --fields, --member_count, --member_fields, --member_filter, --member_sort, --member_sortBy, --member_sortOrder, --member_startIndex, --members, --organization_fields, --organization_memberships, --organization_paid_accounts, --organizations

## `coily ops trello enterprises get-enterprises-id-admins`

Get Enterprise admin Members

Flags: --fields

## `coily ops trello enterprises get-enterprises-id-auditlog`

Get auditlog data for an Enterprise

## `coily ops trello enterprises get-enterprises-id-claimable-organizations`

Get ClaimableOrganizations of an Enterprise

Flags: --activeSince, --cursor, --inactiveSince, --limit, --name

## `coily ops trello enterprises get-enterprises-id-members`

Get Members of Enterprise

Flags: --board_fields, --count, --fields, --filter, --organization_fields, --sort, --sortBy, --sortOrder, --startIndex

## `coily ops trello enterprises get-enterprises-id-members-idmember`

Get a Member of Enterprise

Flags: --board_fields, --fields, --organization_fields

## `coily ops trello enterprises get-enterprises-id-organizations`

Get Organizations of an Enterprise

Flags: --count, --fields, --filter, --startIndex

## `coily ops trello enterprises get-enterprises-id-organizations-bulk-id-organizations`

Bulk accept a set of organizations to an Enterprise.

## `coily ops trello enterprises get-enterprises-id-pending-organizations`

Get PendingOrganizations of an Enterprise

Flags: --activeSince, --inactiveSince

## `coily ops trello enterprises get-enterprises-id-signupurl`

Get signupUrl for Enterprise

Flags: --authenticate, --confirmationAccepted, --returnUrl, --tosAccepted

## `coily ops trello enterprises get-enterprises-id-transferrable-bulk-id-organizations`

Get a bulk list of organizations that can be transferred to an enterprise.

## `coily ops trello enterprises get-enterprises-id-transferrable-organization-id-organization`

Get whether an organization can be transferred to an enterprise.

## `coily ops trello enterprises get-users-id`

Get Users of an Enterprise

Flags: --activeSince, --admin, --collaborator, --cursor, --deactivated, --inactiveSince, --licensed, --managed, --search

## `coily ops trello enterprises post-enterprises-id-tokens`

Create an auth Token for an Enterprise.

Flags: --expiration

## `coily ops trello enterprises put-enterprises-id-admins-idmember`

Update Member to be admin of Enterprise

## `coily ops trello enterprises put-enterprises-id-enterprise-join-request-bulk`

Decline enterpriseJoinRequests from one organization or a bulk list of organizations.

Flags: --idOrganizations

## `coily ops trello enterprises put-enterprises-id-members-idmember-licensed`

Update a Member's licensed status

Flags: --value

## `coily ops trello enterprises put-enterprises-id-organizations`

Transfer an Organization to an Enterprise.

Flags: --idOrganization

## `coily ops trello labels delete-labels-id`

Delete a Label

## `coily ops trello labels get-labels-id`

Get a Label

Flags: --fields

## `coily ops trello labels post-labels`

Create a Label

Flags: --color, --idBoard, --name

## `coily ops trello labels put-labels-id`

Update a Label

Flags: --color, --name

## `coily ops trello labels put-labels-id-field`

Update a field on a label

Flags: --value

## `coily ops trello lists get-lists-id`

Get a List

Flags: --fields

## `coily ops trello lists get-lists-id-actions`

Get Actions for a List

Flags: --filter

## `coily ops trello lists get-lists-id-board`

Get the Board a List is on

Flags: --fields

## `coily ops trello lists get-lists-id-cards`

Get Cards in a List

## `coily ops trello lists post-lists`

Create a new List

Flags: --idBoard, --idListSource, --name, --pos

## `coily ops trello lists post-lists-id-archiveallcards`

Archive all Cards in List

## `coily ops trello lists post-lists-id-moveallcards`

Move all Cards in List

Flags: --idBoard, --idList

## `coily ops trello lists put-id-idboard`

Move List to Board

Flags: --value

## `coily ops trello lists put-lists-id`

Update a List

Flags: --closed, --idBoard, --name, --pos, --subscribed

## `coily ops trello lists put-lists-id-closed`

Archive or unarchive a list

Flags: --value

## `coily ops trello lists put-lists-id-field`

Update a field on a List

Flags: --value

## `coily ops trello members delete-members-id-boardbackgrounds-idbackground`

Delete a Member's custom Board background

## `coily ops trello members delete-members-id-boardstars-idstar`

Delete Star for Board

## `coily ops trello members delete-members-id-customboardbackgrounds-idbackground`

Delete custom Board Background of Member

## `coily ops trello members delete-members-id-customstickers-idsticker`

Delete a Member's custom Sticker

## `coily ops trello members delete-members-id-savedsearches-idsearch`

Delete a saved search

## `coily ops trello members get-members-id`

Get a Member

Flags: --actions, --boardBackgrounds, --boardStars, --boards, --boardsInvited, --boardsInvited_fields, --cards, --customBoardBackgrounds, --customEmoji, --customStickers, --fields, --notifications, --organization_fields, --organization_paid_account, --organizations, --organizationsInvited, --organizationsInvited_fields, --paid_account, --savedSearches, --tokens

## `coily ops trello members get-members-id-actions`

Get a Member's Actions

Flags: --filter

## `coily ops trello members get-members-id-boardbackgrounds`

Get Member's custom Board backgrounds

Flags: --filter

## `coily ops trello members get-members-id-boardbackgrounds-idbackground`

Get a boardBackground of a Member

Flags: --fields

## `coily ops trello members get-members-id-boards`

Get Boards that Member belongs to

Flags: --fields, --filter, --lists, --organization, --organization_fields

## `coily ops trello members get-members-id-boardsinvited`

Get Boards the Member has been invited to

Flags: --fields

## `coily ops trello members get-members-id-boardstars`

Get a Member's boardStars

## `coily ops trello members get-members-id-boardstars-idstar`

Get a boardStar of Member

## `coily ops trello members get-members-id-cards`

Get Cards the Member is on

Flags: --filter

## `coily ops trello members get-members-id-customboardbackgrounds`

Get a Member's custom Board Backgrounds

## `coily ops trello members get-members-id-customboardbackgrounds-idbackground`

Get custom Board Background of Member

## `coily ops trello members get-members-id-customemoji`

Get a Member's customEmojis

## `coily ops trello members get-members-id-customstickers`

Get Member's custom Stickers

## `coily ops trello members get-members-id-customstickers-idsticker`

Get a Member's custom Sticker

Flags: --fields

## `coily ops trello members get-members-id-field`

Get a field on a Member

## `coily ops trello members get-members-id-notification-channel-settings`

Get a Member's notification channel settings

## `coily ops trello members get-members-id-notification-channel-settings-channel`

Get blocked notification keys of Member on this channel

## `coily ops trello members get-members-id-notifications`

Get Member's Notifications

Flags: --before, --display, --entities, --fields, --filter, --limit, --memberCreator, --memberCreator_fields, --page, --read_filter, --since

## `coily ops trello members get-members-id-organizations`

Get Member's Organizations

Flags: --fields, --filter, --paid_account

## `coily ops trello members get-members-id-organizationsinvited`

Get Organizations a Member has been invited to

Flags: --fields

## `coily ops trello members get-members-id-savedsearches`

Get Member's saved searched

## `coily ops trello members get-members-id-savedsearches-idsearch`

Get a saved search

## `coily ops trello members get-members-id-tokens`

Get Member's Tokens

Flags: --webhooks

## `coily ops trello members membersidavatar`

Create Avatar for Member

Flags: --file

## `coily ops trello members membersidcustomboardbackgrounds-1`

Create a new custom Board Background

Flags: --file

## `coily ops trello members membersidcustomemojiidemoji`

Get a Member's custom Emoji

Flags: --fields

## `coily ops trello members post-members-id-boardbackgrounds-1`

Upload new boardBackground for Member

Flags: --file

## `coily ops trello members post-members-id-boardstars`

Create Star for Board

Flags: --idBoard, --pos

## `coily ops trello members post-members-id-customemoji`

Create custom Emoji for Member

Flags: --file, --name

## `coily ops trello members post-members-id-customstickers`

Create custom Sticker for Member

Flags: --file

## `coily ops trello members post-members-id-onetimemessagesdismissed`

Dismiss a message for Member

Flags: --value

## `coily ops trello members post-members-id-savedsearches`

Create saved Search for Member

Flags: --name, --pos, --query

## `coily ops trello members put-members-id`

Update a Member

Flags: --avatarSource, --bio, --fullName, --initials, --prefs/colorBlind, --prefs/locale, --prefs/minutesBetweenSummaries, --username

## `coily ops trello members put-members-id-boardbackgrounds-idbackground`

Update a Member's custom Board background

Flags: --brightness, --tile

## `coily ops trello members put-members-id-boardstars-idstar`

Update the position of a boardStar of Member

Flags: --pos

## `coily ops trello members put-members-id-customboardbackgrounds-idbackground`

Update custom Board Background of Member

Flags: --brightness, --tile

## `coily ops trello members put-members-id-notification-channel-settings-channel-blocked-keys`

Update blocked notification keys of Member on a channel

Flags: --body

## `coily ops trello members put-members-id-notification-channel-settings-channel-blocked-keys-put`

Update blocked notification keys of Member on a channel

Flags: --body

## `coily ops trello members put-members-id-notification-channel-settings-channel-blocked-keys-put3`

Update blocked notification keys of Member on a channel

## `coily ops trello members put-members-id-savedsearches-idsearch`

Update a saved search

Flags: --name, --pos, --query

## `coily ops trello notifications get-notifications-id`

Get a Notification

Flags: --board, --board_fields, --card, --card_fields, --display, --entities, --fields, --list, --member, --memberCreator, --memberCreator_fields, --member_fields, --organization, --organization_fields

## `coily ops trello notifications get-notifications-id-board`

Get the Board a Notification is on

Flags: --fields

## `coily ops trello notifications get-notifications-id-card`

Get the Card a Notification is on

Flags: --fields

## `coily ops trello notifications get-notifications-id-field`

Get a field of a Notification

## `coily ops trello notifications get-notifications-id-list`

Get the List a Notification is on

Flags: --fields

## `coily ops trello notifications get-notifications-id-membercreator`

Get the Member who created the Notification

Flags: --fields

## `coily ops trello notifications get-notifications-id-organization`

Get a Notification's associated Organization

Flags: --fields

## `coily ops trello notifications notificationsidmember`

Get the Member a Notification is about (not the creator)

Flags: --fields

## `coily ops trello notifications post-notifications-all-read`

Mark all Notifications as read

Flags: --ids, --read

## `coily ops trello notifications put-notifications-id`

Update a Notification's read status

Flags: --unread

## `coily ops trello notifications put-notifications-id-unread`

Update Notification's read status

Flags: --value

## `coily ops trello organizations delete-organizations-id`

Delete an Organization

## `coily ops trello organizations delete-organizations-id-logo`

Delete Logo for Organization

## `coily ops trello organizations delete-organizations-id-members`

Remove a Member from an Organization

## `coily ops trello organizations delete-organizations-id-prefs-associateddomain`

Remove the associated Google Apps domain from a Workspace

## `coily ops trello organizations delete-organizations-id-prefs-orginviterestrict`

Delete the email domain restriction on who can be invited to the Workspace

## `coily ops trello organizations delete-organizations-id-tags-idtag`

Delete an Organization's Tag

## `coily ops trello organizations get-organizations-id`

Get an Organization

## `coily ops trello organizations get-organizations-id-actions`

Get Actions for Organization

## `coily ops trello organizations get-organizations-id-boards`

Get Boards in an Organization

Flags: --fields, --filter

## `coily ops trello organizations get-organizations-id-exports`

Retrieve Organization's Exports

## `coily ops trello organizations get-organizations-id-field`

Get field on Organization

## `coily ops trello organizations get-organizations-id-members`

Get the Members of an Organization

## `coily ops trello organizations get-organizations-id-memberships`

Get Memberships of an Organization

Flags: --filter, --member

## `coily ops trello organizations get-organizations-id-memberships-idmembership`

Get a Membership of an Organization

Flags: --member

## `coily ops trello organizations get-organizations-id-newbillableguests-idboard`

Get Organizations new billable guests

## `coily ops trello organizations get-organizations-id-plugindata`

Get the pluginData Scoped to Organization

## `coily ops trello organizations get-organizations-id-tags`

Get Tags of an Organization

## `coily ops trello organizations organizations-id-members-idmember-all`

Remove a Member from an Organization and all Organization Boards

## `coily ops trello organizations post-organizations`

Create a new Organization

Flags: --desc, --displayName, --name, --website

## `coily ops trello organizations post-organizations-id-exports`

Create Export for Organizations

Flags: --attachments

## `coily ops trello organizations post-organizations-id-logo`

Update logo for an Organization

Flags: --file

## `coily ops trello organizations post-organizations-id-tags`

Create a Tag in Organization

## `coily ops trello organizations put-organizations-id`

Update an Organization

Flags: --desc, --displayName, --name, --prefs/associatedDomain, --prefs/boardVisibilityRestrict/org, --prefs/boardVisibilityRestrict/private, --prefs/boardVisibilityRestrict/public, --prefs/externalMembersDisabled, --prefs/googleAppsVersion, --prefs/orgInviteRestrict, --prefs/permissionLevel, --website

## `coily ops trello organizations put-organizations-id-members`

Update an Organization's Members

Flags: --email, --fullName, --type

## `coily ops trello organizations put-organizations-id-members-idmember`

Update a Member of an Organization

Flags: --type

## `coily ops trello organizations put-organizations-id-members-idmember-deactivated`

Deactivate or reactivate a member of an Organization

Flags: --value

## `coily ops trello plugins get-plugins-id`

Get a Plugin

## `coily ops trello plugins get-plugins-id-compliance-memberprivacy`

Get Plugin's Member privacy compliance

## `coily ops trello plugins post-plugins-idplugin-listing`

Create a Listing for Plugin

Flags: --body

## `coily ops trello plugins put-plugins-id`

Update a Plugin

## `coily ops trello plugins put-plugins-idplugin-listings-idlisting`

Updating Plugin's Listing

Flags: --body

## `coily ops trello search get-search`

Search Trello

Flags: --board_fields, --board_organization, --boards_limit, --card_attachments, --card_board, --card_fields, --card_list, --card_members, --card_stickers, --cards_limit, --cards_page, --idBoards, --idCards, --idOrganizations, --member_fields, --members_limit, --modelTypes, --organization_fields, --organizations_limit, --partial, --query

## `coily ops trello search get-search-members`

Search for Members

Flags: --idBoard, --idOrganization, --limit, --onlyOrgMembers, --query

## `coily ops trello tokens delete-token`

Delete a Token

## `coily ops trello tokens delete-tokens-token-webhooks-idwebhook`

Delete a Webhook created by Token

## `coily ops trello tokens get-tokens-token`

Get a Token

Flags: --fields, --webhooks

## `coily ops trello tokens get-tokens-token-member`

Get Token's Member

Flags: --fields

## `coily ops trello tokens get-tokens-token-webhooks`

Get Webhooks for Token

## `coily ops trello tokens get-tokens-token-webhooks-idwebhook`

Get a Webhook belonging to a Token

## `coily ops trello tokens post-tokens-token-webhooks`

Create Webhooks for Token

Flags: --callbackURL, --description, --idModel

## `coily ops trello tokens tokenstokenwebhooks-1`

Update a Webhook created by Token

Flags: --callbackURL, --description, --idModel

## `coily ops trello webhooks delete-webhooks-id`

Delete a Webhook

## `coily ops trello webhooks get-webhooks-id`

Get a Webhook

## `coily ops trello webhooks post-webhooks`

Create a Webhook

Flags: --active, --callbackURL, --description, --idModel

## `coily ops trello webhooks put-webhooks-id`

Update a Webhook

Flags: --active, --callbackURL, --description, --idModel

## `coily ops trello webhooks webhooksidfield`

Get a field on a Webhook

## `coily pkg brew`

Scoped wrapper around brew. Mirrors brew's argv shape.

## `coily pkg bun`

Pass-through to bun with argv validation + audit log.

## `coily pkg bundle`

Pass-through to bundle with argv validation + audit log.

## `coily pkg cargo`

Pass-through to cargo with argv validation + audit log.

## `coily pkg gem`

Pass-through to gem with argv validation + audit log.

## `coily pkg glama instances get-v1-instances`

Hosted MCP server instances

## `coily pkg glama servers get-v1-attributes`

MCP server attributes

## `coily pkg glama servers get-v1-servers`

GET /v1/servers

Flags: --after, --first, --query

## `coily pkg glama servers get-v1-servers-by-namespace-by-slug`

Retrieve the MCP server details by its unique identifier.

## `coily pkg glama telemetry post-v1-telemetry-usage`

Send MCP tool usage data to the server.

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

## `coily pkg skillsmp ai-search`

GET /ai-search - semantic search across skills.

Flags: --output, --query

## `coily pkg skillsmp search`

GET /search - keyword search across skills.

Flags: --limit, --output, --page, --query, --sort-by

## `coily pkg uv`

Pass-through to uv with argv validation + audit log.

## `coily pkg yarn`

Pass-through to yarn with argv validation + audit log.

## `coily session clear`

Remove the per-session sentinel. No-op if absent.

## `coily session show`

Print the active profile and (phase-2) the would-be strictest axis tiers.

## `coily session use`

Record the active lockdown profile for this Claude Code session.

## `coily setup`

Run the post-upgrade rituals: completion, lockdown re-baseline, and user hook.

Flags: --lockdown-root, --skip-completion, --skip-host-bootstrap, --skip-lockdown, --skip-skills, --skip-user-hook

## `coily ssh`

Free-form passthrough to a configured host alias.

## `coily systemctl daemon-reload`

Run systemctl daemon-reload.

## `coily systemctl disable`

Disable <unit>.

## `coily systemctl enable`

Enable <unit>.

## `coily systemctl restart`

Restart <unit>.

## `coily systemctl start`

Start <unit>.

## `coily systemctl status`

Print systemctl status of <unit>.

## `coily systemctl stop`

Stop <unit>.

## `coily tailscale`

Pass-through to tailscale with argv validation + audit log.

## `coily upgrade`

Self-update via brew (coilysiren tap, per-repo or umbrella).

Flags: --dry

## `coily version`

Print the build version and exit.

## `coily whoami`

Print the authenticated identity coily sees across aws, kubectl, and gh.
