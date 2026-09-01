---
name: agwctl-usage
description: Resolve and call MCP tools through the agwctl shell client instead of a direct agentgateway MCP connection. Use when agwctl is available via Bash and tool schemas should not be loaded into session context.
---

# agwctl usage

`agwctl` talks to the local agentgateway (`http://127.0.0.1:8083/mcp`, 8 targets,
~70 tools). Resolve tools at call time; never keep schemas in context.

## Availability

The binary lives at `~/go/bin/agwctl`; the PATH entry comes from `~/.zshrc`
(added 2026-09-02). Harnesses launched from a GUI shell may not have it, so
fall back to the full path. The direct `agentgateway` MCP entry was dropped
from every harness the same day (`mcp-sync.py --drop agentgateway`); this CLI
is the only access path, and `mcp__agentgateway__*` tools in old sessions are
stale.

## The three-step flow

```bash
# 1. find candidate tools (top 5, ~100 tokens)
agwctl tools search "search the web"

# 2. get the schema for the one you want
agwctl tools describe brave-search_brave_web_search

# 3. call it, shaped for the context budget
agwctl call brave-search_brave_web_search --arg query="mcp gateway" \
    --jq '.content[0].text' --max-chars 4000
```

`tools search` results carry a lexical score. Higher is better; ties break by
name. `--limit N` widens the net; an empty result means no tool fits.

## Calling

- `--arg k=v` repeatable; values parse as JSON first (`5` number, `true` bool,
  `"x"` string, otherwise plain string)
- `--json '{"query":"..."}'` or `--json @request.json` for a full object
  (mutually exclusive with `--arg`)
- `--jq EXPR` runs over the raw CallToolResult; `--jq .` dumps the whole
  result as JSON for scripts; skipped when the tool returns isError
- `--max-chars` (default 20000) truncates text output with a marker; `0`
  disables; JSON output is never cut
- `--out file.json` writes the full result (raw CallToolResult JSON unless
  `--jq` is set) and prints only the path to stdout

## Exit codes

| code | meaning |
|---|---|
| 0 | success (empty search result is also 0) |
| 1 | tool returned isError; the error text is on stdout, read and self-correct |
| 2 | usage error |
| 3 | gateway unreachable or initialize failed |
| 4 | timeout (`--timeout`, default 300s) |
| 5 | transport/protocol failure; retry or report |

## Multi-step workflows

Polling, pagination, fan-out: write ONE bash script and run it in ONE Bash
tool call, so intermediate results never reach the LLM. See
`examples/poll-query.sh` in the repo. Batch requests over one session with
`--stdin` (JSONL in, JSONL out).

## Discipline

- Pipe big results through `--jq`, or land them with `--out` and process with
  jq/python; do not paste 500 KB into context.
- Tools rarely declare outputSchema. Probe once (`call ... --max-chars 300`)
  to learn the shape before writing jq.
- Exit 1 with content on stdout means the tool itself reported an error; read
  it and adjust arguments. Exit 3/4/5 are transport problems; retry or report.
