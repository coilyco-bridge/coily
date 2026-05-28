#!/usr/bin/env bash
set -euo pipefail

# Usage: sync-lockdown.sh <canon-dir> <tag> <catalog-yaml>
# Fans the canonical .claude lockdown files out to every non-archived
# catalog repo via the forgejo Contents API. Idempotent and loop-safe.
# Design + rationale: coilysiren/agentic-os-kai#457, coilysiren/agentic-os#92.

CANON_DIR=${1:?canon dir required}
TAG=${2:?tag required}
CATALOG=${3:?catalog yaml path required}

FORGEJO_BASE_URL=${FORGEJO_BASE_URL:-https://forgejo.coilysiren.me}
: "${FORGEJO_TOKEN:?FORGEJO_TOKEN required (org-scoped PAT)}"

FILES=(".claude/settings.json" ".claude/lockdown-deny.sh")
# [skip ci] keeps the bot commits from re-triggering downstream pipelines,
# matching bump-formula. Tradeoff: each repo's mirror-to-github is also
# skipped, so GitHub lags until that repo's next non-skip push.
COMMIT_MSG="lockdown: sync to coily ${TAG} [skip ci]"

ensure_tool() {
  command -v "$1" >/dev/null 2>&1 && return 0
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -qq && apt-get install -y -qq "$1"
  elif command -v apk >/dev/null 2>&1; then
    apk add --no-cache "$1"
  else
    echo "::error::cannot install $1 (no apt-get or apk)" >&2
    exit 1
  fi
}

ensure_tool jq
ensure_tool curl
if ! command -v yq >/dev/null 2>&1; then
  curl -fsSL -o /usr/local/bin/yq \
    https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64
  chmod +x /usr/local/bin/yq
fi

for f in "${FILES[@]}"; do
  [ -f "${CANON_DIR}/${f}" ] || { echo "::error::missing ${CANON_DIR}/${f}" >&2; exit 1; }
done

api() {
  # api <method> <path> [json-body-file] -> writes body to /tmp/resp, echoes HTTP code
  local method=$1 path=$2 body=${3:-}
  local args=(-sS -o /tmp/resp -w '%{http_code}' -X "$method"
    -H "Authorization: token ${FORGEJO_TOKEN}" -H "Content-Type: application/json")
  [ -n "$body" ] && args+=(-d "@${body}")
  curl "${args[@]}" "${FORGEJO_BASE_URL}/api/v1/${path}"
}

repo_exists() {
  [ "$(api GET "repos/$1")" = "200" ]
}

put_file() {
  # put_file <owner/repo> <path> <local-file> [<sha>]
  local repo=$1 path=$2 local=$3 sha=${4:-}
  local b64 payload
  b64=$(base64 -w0 <"$local")
  payload=$(jq -n --arg m "$COMMIT_MSG" --arg c "$b64" --arg b main --arg s "$sha" \
    'if $s == "" then {message:$m, content:$c, branch:$b}
     else {message:$m, content:$c, branch:$b, sha:$s} end' >/tmp/put.json && echo /tmp/put.json)
  api PUT "repos/${repo}/contents/${path}" "$payload"
}

fails=0
mapfile -t REPOS < <(yq '.nodes[] | select(.archived == false) | .org + "/" + .name' "$CATALOG")

for repo in "${REPOS[@]}"; do
  for f in "${FILES[@]}"; do
    local_file="${CANON_DIR}/${f}"
    code=$(api GET "repos/${repo}/contents/${f}?ref=main")
    case "$code" in
      200)
        if jq -r '.content' </tmp/resp | base64 -d | cmp -s - "$local_file"; then
          echo "skip (current): ${repo}/${f}"
          continue
        fi
        sha=$(jq -r '.sha' </tmp/resp)
        pcode=$(put_file "$repo" "$f" "$local_file" "$sha")
        ;;
      404)
        if ! repo_exists "$repo"; then
          echo "skip (not on forgejo): ${repo}"
          break
        fi
        pcode=$(put_file "$repo" "$f" "$local_file")
        ;;
      423)
        echo "skip (archived): ${repo}"
        break
        ;;
      *)
        echo "::warning::GET ${repo}/${f} -> HTTP ${code}"; cat /tmp/resp
        fails=$((fails + 1))
        continue
        ;;
    esac
    case "${pcode:-}" in
      200 | 201) echo "synced: ${repo}/${f}" ;;
      423) echo "skip (archived): ${repo}"; break ;;
      *) echo "::warning::PUT ${repo}/${f} -> HTTP ${pcode}"; cat /tmp/resp; fails=$((fails + 1)) ;;
    esac
  done
done

echo "sync-lockdown: ${#REPOS[@]} catalog repos processed, ${fails} unexpected failures"
[ "$fails" -eq 0 ]
