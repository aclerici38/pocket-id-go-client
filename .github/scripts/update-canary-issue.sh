#!/usr/bin/env bash
set -euo pipefail

# Renders the upstream API canary dashboard issue (created locked on first run) with the
# API diff of upstream main against the latest release. The issue is assigned to
# CANARY_ASSIGNEE while the client fails to build or the diff has breaking changes.
#
# Required env: GH_TOKEN, CANARY_LABEL, CANARY_ASSIGNEE, UPSTREAM_SHA, BUILD (the
#               generate step's outcome), RUN_URL; reads openapi.json from the working directory.

: "${CANARY_LABEL:?}" "${CANARY_ASSIGNEE:?}" "${UPSTREAM_SHA:?}" "${BUILD:?}" "${RUN_URL:?}"

FENCE='```'
BASELINE="$(gh release view --json tagName --jq .tagName)"

# Issue bodies max out at 65536 characters; keep the inline diff well under that.
cap() { # file limit
  if [ "$(wc -c < "$1")" -le "$2" ]; then cat "$1"; return; fi
  head -c "$2" "$1" | sed '$d'
  echo "... truncated"
}

if [ "$BUILD" != success ]; then
  FAILING=true
  RESULTS="❌ **Generating or building the client failed**, see the [run](${RUN_URL})."
else
  dir="$(mktemp -d)"
  gh api "repos/${GITHUB_REPOSITORY}/contents/openapi.json?ref=${BASELINE}" \
    -H "Accept: application/vnd.github.raw" > "$dir/$BASELINE.json"
  cp openapi.json "$dir/main.json"
  cd "$dir"

  git diff --no-index "$BASELINE.json" main.json > api.diff || true
  if oasdiff breaking --fail-on ERR --color never "$BASELINE.json" main.json > breaking.txt; then
    FAILING=false
  else
    FAILING=true
  fi

  if [ ! -s api.diff ]; then
    RESULTS="✅ **No API changes** since \`${BASELINE}\`."
  else
    if [ "$FAILING" = true ]; then HEADING="❌ **Breaking changes** since \`${BASELINE}\`"
    else HEADING="⚠️ **API changes** since \`${BASELINE}\`, none breaking"
    fi
    RESULTS="${HEADING}
${FENCE}
$(cat breaking.txt)
${FENCE}

<details><summary>API diff</summary>

${FENCE}diff
$(cap api.diff 50000)
${FENCE}
</details>"
  fi
fi

BODY="## Upstream API canary

|          |   |
| -------- | - |
| upstream | [\`pocket-id@${UPSTREAM_SHA:0:7}\`](https://github.com/pocket-id/pocket-id/commit/${UPSTREAM_SHA}) (\`main\`) |
| baseline | [\`${BASELINE}\`](https://github.com/${GITHUB_REPOSITORY}/releases/tag/${BASELINE}) |
| last run | $(date -u +%Y-%m-%dT%H:%M:%SZ) ([run](${RUN_URL})) |

${RESULTS}

---
_Automated hourly by \`.github/workflows/upstream-canary.yaml\`; this issue is locked, do not edit._
_While the client fails to build or has breaking changes, @${CANARY_ASSIGNEE} is assigned._"

NUM="$(gh issue list --label "$CANARY_LABEL" --state all --limit 1 --json number --jq '.[0].number // empty')"
if [ -z "$NUM" ]; then
  gh label create "$CANARY_LABEL" --color BFD4F2 --description "Upstream API canary dashboard" 2>/dev/null || true
  NUM="$(gh issue create --title "Upstream API canary" --label "$CANARY_LABEL" --body "$BODY" | grep -oE '[0-9]+$')"
  gh issue lock "$NUM" 2>/dev/null || true
else
  gh issue edit "$NUM" --body "$BODY"
  gh issue reopen "$NUM" 2>/dev/null || true
fi

if [ "$FAILING" = true ]; then
  gh issue edit "$NUM" --add-assignee "$CANARY_ASSIGNEE"
else
  gh issue edit "$NUM" --remove-assignee "$CANARY_ASSIGNEE"
fi
