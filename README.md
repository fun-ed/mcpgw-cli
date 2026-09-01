# mcpgw-cli

agentgateway 的 CLI client（binary 名稱 `agwctl`）。把 harness 對 MCP 的直接依賴
換成一個 shell 指令：tool schema 不再常駐 session context，呼叫當下才 resolve。

```text
Copilot CLI / Claude Code / Kiro / Codex / Devin
        │
        │ Bash
        ▼
      agwctl
        │ initialize / tools/list / tools/call
        ▼
   agentgateway v1.5.0+
        │
        └─ 8 個 MCP targets（Hindsight、Exa、OpenViking、DeepWiki、Tavily、
           Ref Context、brave-search、duckduckgo-search）
```

## 適配範圍

以 **agentgateway v1.5.0+** 的 legacy stateful streamable HTTP 為主：
`initialize` handshake、`mcp-session-id` header、session DELETE。對活 gateway
實測協商 `protocolVersion 2025-06-18`。MCP 2026-07-28 的 stateless
`server/discover` 模式要等 gateway 端支援後另行適配，見 `PLAN.md` 風險表。

## 安裝

```bash
# Go 1.26.x，由 .tool-versions（mise）釘住
go build -o ~/go/bin/agwctl ./cmd/agwctl
```

Gateway 本機無 auth，`agwctl` 不需要任何 token。endpoint 預設
`http://127.0.0.1:8083/mcp`，可用 `--url` 或 `AGWCTL_URL` 覆蓋。

## 用法：find → describe → call

```bash
agwctl tools search "search the web"          # top 5，~100 tokens
agwctl tools describe brave-search_web_search # 只印這個 tool 的 inputSchema
agwctl call brave-search_brave_web_search --arg query="mcp gateway" \
    --jq '.content[0].text' --max-chars 4000  # 只要結果，不要整坨
```

多步流程（輪詢、分頁）寫成一段 shell script 在一個 Bash tool call 內跑完，
見 `examples/poll-query.sh` 與 `skills/agwctl-usage/SKILL.md`。

## 指令

| 指令 | 用途 |
|---|---|
| `tools list [--target T] [--json]` | 名稱 + 一行描述，不帶 schema |
| `tools search QUERY [--limit N]` | lexical 排序 top N（tool-list cache，10 分鐘 TTL） |
| `tools describe NAME` | 該 tool 的 name、description、inputSchema |
| `call NAME [--arg k=v] [--json @f] [--jq EXPR] [--max-chars N] [--out F]` | 呼叫一個 tool |
| `call --stdin` | JSONL batch：多請求、單一 session、一次 initialize |

## Exit codes

| code | 意義 |
|---|---|
| 0 | 成功（search 沒命中也是 0） |
| 1 | tool 回 `isError`，錯誤內容已印在 stdout |
| 2 | 參數用法錯誤 |
| 3 | gateway 不可達或 initialize 失敗 |
| 4 | 逾時 |
| 5 | transport / protocol 錯誤 |
| 6 | `doctor --expect-targets` 不符（M3） |

## 量測（2026-09-01，8 targets / 70 tools）

| 量項 | 數字 |
|---|---|
| 改善前：完整 schema + 描述 | ≈ 18.6k tokens / session |
| `tools list`（無 schema） | 6,653 bytes ≈ 1.7k tokens，省約 91% |
| `tools search` top 5 | < 150 tokens；快取命中 0.024s |
| 單次執行 | 約 3s（initialize 0.5s + gateway fanout floor 約 0.5s/請求） |

## 文件地圖

| 檔案 | 內容 |
|---|---|
| `PLAN.md` | **SOT**：設計、契約、量測、里程碑 |
| `AGENTS.md` | AI agent 的 repo 操作守則 |
| `docs/development.md` | 開發準則（worktree、依賴、code style） |
| `docs/maintenance.md` | 維護準則（例行檢查、升級、release、skills 維護） |
| `docs/quality.md` | Review / Eval / Test 準則 |
| `docs/pr-handoff.md` | 交接 PR 準則 |
| `skills/` | 內建 agent skills（agwctl-usage、agwctl-maintenance） |

Go module：`github.com/fun-ed/mcpgw-cli`；binary 維持 `agwctl`（文件與契約用它）。
