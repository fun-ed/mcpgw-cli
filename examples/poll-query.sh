#!/usr/bin/env bash
# Template: run a multi-step tool workflow inside ONE Bash tool call so the
# LLM only sees the final result. Replace the tool names, argument keys and
# the .state jq path with the real ones (probe once with agwctl call first).
#
# Usage: ID=abc123 ./examples/poll-query.sh
set -euo pipefail

state=""
for i in $(seq 1 10); do
  state=$(agwctl call target_query_status --arg id="$ID" --json \
            | jq -r '.content[0].text | fromjson | .state' || true)
  [ "$state" = "done" ] && break
  sleep 5
done
[ "$state" = "done" ] || { echo "query $ID did not finish" >&2; exit 1; }
agwctl call target_get_query_result --arg id="$ID" --max-chars 8000
