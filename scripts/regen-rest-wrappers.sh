#!/usr/bin/env bash
# Regenerate every coily REST wrapper from its upstream OpenAPI spec via
# scripts/openapi-to-coily.py. Each surface pins the exact generator arguments
# that produced the committed cmd/coily/<pkg>/<pkg>_generated.go, so a re-run
# refreshes the wrapper against spec drift without hand-editing.
#
# The generator folds coily's --query/--output projection rail (restfmt,
# coilyco-bridge/coily#46) into every surface natively, so there is no
# hand-written per-surface wrapper to keep in sync.
#
# Network access to the upstream specs is required. Run from the repo root:
#   bash scripts/regen-rest-wrappers.sh
set -euo pipefail

cd "$(dirname "$0")/.."
GEN=scripts/openapi-to-coily.py

python3 "$GEN" \
  --spec https://glama.ai/api/mcp/openapi.json \
  --pkg glama \
  --usage "Glama MCP API directory + telemetry." \
  --base-url https://glama.ai/api/mcp \
  --auth-mode bearer-when-required \
  --ssm-param /glama/api-key \
  --mount pkg \
  --out cmd/coily/glama/glama_generated.go

python3 "$GEN" \
  --spec https://developer.atlassian.com/cloud/trello/swagger.v3.json \
  --pkg trello \
  --usage "Trello REST API (key+token auth)." \
  --base-url https://api.trello.com/1 \
  --auth-mode trello-keytoken \
  --ssm-param /trello/api-key \
  --ssm-param-aux /trello/token \
  --mount ops \
  --out cmd/coily/trello/trello_generated.go

python3 "$GEN" \
  --spec https://raw.githubusercontent.com/getsentry/sentry/master/api-docs/openapi.json \
  --pkg sentry \
  --usage "Sentry Public API (bearer auth)." \
  --base-url https://us.sentry.io \
  --auth-mode bearer-required \
  --ssm-param /sentry/api-key \
  --mount ops \
  --out cmd/coily/sentry/sentry_generated.go

python3 "$GEN" \
  --spec https://raw.githubusercontent.com/discord/discord-api-spec/main/specs/openapi.json \
  --pkg discord \
  --usage "Discord HTTP API (bot auth)." \
  --base-url https://discord.com/api/v10 \
  --auth-mode discord-bot \
  --ssm-param /discord/bot-token \
  --mount ops \
  --out cmd/coily/discord/discord_generated.go

echo "regenerated glama, trello, sentry, discord"
