#!/usr/bin/env bash
set -euo pipefail

# Renders the upstream API canary dashboard issue from a canary run's results,
# creating it (locked) on first run. The hourly heartbeat for unchanged runs
# only touches the "last checked" row and is done inline in the workflow.
#
# Required env: GH_TOKEN, CANARY_LABEL, CANARY_ASSIGNEE, STATUS (unchanged|changed|failing),
#               UPSTREAM_SHA, BASELINE, RUN_URL
# Optional env: API_DIR (holds breaking.txt and api.diff; absent if generation failed),
#               ARTIFACT_URL

: "${CANARY_LABEL:?}" "${CANARY_ASSIGNEE:?}" "${STATUS:?}" "${UPSTREAM_SHA:?}" "${BASELINE:?}" "${RUN_URL:?}"

NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
FENCE='```'

# Issue bodies max out at 65536 characters; keep the inline diff well under that.
cap() { # file limit
  if [ "$(wc -c < "$1")" -le "$2" ]; then cat "$1"; return; fi
  head -c "$2" "$1" | sed '$d'
  echo "... truncated, full diff in the run artifact"
}

case "$STATUS" in
  unchanged) BADGE="✅ **UNCHANGED**: matches \`${BASELINE}\`" ;;
  changed)   BADGE="⚠️ **CHANGED**: differs from \`${BASELINE}\`, no breaking changes" ;;
  failing)   BADGE="❌ **FAILING**: breaking changes or generation/build failed" ;;
esac

if [ -f "${API_DIR:-}/api.diff" ]; then
  RESULTS="### Breaking changes
${FENCE}
$(cat "$API_DIR/breaking.txt")
${FENCE}

<details><summary>API diff vs <code>${BASELINE}</code>${ARTIFACT_URL:+ (<a href=\"${ARTIFACT_URL}\">full artifact</a>)}</summary>

${FENCE}diff
$(cap "$API_DIR/api.diff" 50000)
${FENCE}
</details>"
else
  RESULTS="Client generation or build failed, so there is nothing to compare. See the [run](${RUN_URL})."
fi

BODY="## Upstream API canary

<!-- state:tested=${UPSTREAM_SHA}@${BASELINE} state:status=${STATUS} -->

${BADGE}

|              |   |
| ------------ | - |
| upstream     | [\`pocket-id@${UPSTREAM_SHA:0:7}\`](https://github.com/pocket-id/pocket-id/commit/${UPSTREAM_SHA}) (\`main\`) |
| baseline     | [\`${BASELINE}\`](https://github.com/${GITHUB_REPOSITORY}/releases/tag/${BASELINE}) |
| last tested  | ${NOW} ([run](${RUN_URL})) |
| last checked | ${NOW} |

${RESULTS}

---
_Automated hourly by \`.github/workflows/upstream-canary.yaml\`; this issue is locked, do not edit._
_Runs only when upstream \`main\` or the latest release changes; \`last checked\` updates every hour regardless._
_While failing, @${CANARY_ASSIGNEE} is assigned; the assignment clears on the next run that isn't failing._"

NUM="$(gh issue list --label "$CANARY_LABEL" --state all --limit 1 --json number --jq '.[0].number // empty')"
if [ -z "$NUM" ]; then
  gh label create "$CANARY_LABEL" --color BFD4F2 --description "Upstream API canary dashboard" 2>/dev/null || true
  NUM="$(gh issue create --title "Upstream API canary" --label "$CANARY_LABEL" --body "$BODY" | grep -oE '[0-9]+$')"
  gh issue lock "$NUM" 2>/dev/null || true
else
  gh issue edit "$NUM" --body "$BODY"
  gh issue reopen "$NUM" 2>/dev/null || true
fi

if [ "$STATUS" = failing ]; then
  gh issue edit "$NUM" --add-assignee "$CANARY_ASSIGNEE"
else
  gh issue edit "$NUM" --remove-assignee "$CANARY_ASSIGNEE"
fi
