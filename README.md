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
`initialize` handshake、`mcp-session-id` header、session DELETE。實測協商
`protocolVersion` 在 `2025-06-18` 與 `2025-11-25` 之間（皆 legacy stateful）。

gateway 本體已支援 MCP 2026-07-28（stateless `server/discover`），但
multiplexing 模式下協商取**全部 upstream 的版本交集**：任何一個 legacy
upstream 都把整體壓在 legacy。agwctl 跟著協商結果走，client 端不單方面升級。

## 安裝

```bash
# Go 1.26.x，由 .tool-versions（mise）釘住
go build -o ~/go/bin/agwctl ./cmd/agwctl
```

安裝後在 `/usr/local/bin` 放一個 symlink（2026-09-02 已做）：
`sudo ln -s "$HOME/go/bin/agwctl" /usr/local/bin/agwctl`。GUI 啟動的 harness
也找得到 bare `agwctl`；rebuild 仍只寫 `~/go/bin/agwctl`，symlink 自動跟上。

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
| `doctor [--expect-targets N] [--json]` | 可達性、initialize 延遲、每 target tool 數、異常提示；永遠活取 |

## Exit codes

| code | 意義 |
|---|---|
| 0 | 成功（search 沒命中也是 0） |
| 1 | tool 回 `isError`，錯誤內容已印在 stdout |
| 2 | 參數用法錯誤 |
| 3 | gateway 不可達或 initialize 失敗 |
| 4 | 逾時 |
| 5 | transport / protocol 錯誤 |
| 6 | `doctor --expect-targets` 不符 |

## Benchmark（2026-09-01 實測 + cutover 記錄）

| 情境 | schema 進 session context | 中間結果 |
|---|---|---|
| 直接 MCP 連線（cutover 前） | 70 tools 完整 schema ≈ **18.6k tokens**，每個 turn 重複背 | 每步 tool call 的完整回應都回 LLM |
| agwctl（cutover 後） | **0**：harness 只知道一個 bash 指令 | `--jq` / `--max-chars` / `--out` 控制，只有最終結果回 LLM |

| 操作 | 成本 | 備註 |
|---|---|---|
| `tools search` top 5 | < 150 tokens；快取命中 0.024s | 快取 TTL 10 分鐘 |
| `tools list`（一次性） | 6,653 bytes ≈ 1.7k tokens | 約省 91% |
| `tools describe` | 只付該 tool 的 schema | 唯一需要 schema 的時候 |
| `call --jq/--max-chars` | 預設輸出上界 20,000 chars | deepwiki 3k 字答案只取需要的段 |
| 單次執行延遲 | 約 3s | initialize 0.5s + gateway fanout floor 約 0.5s/請求 |

### Cutover 記錄（2026-09-01）

- **Kiro**（acpx dispatch，session 無 gateway MCP）：只用 shell + agwctl 完成
  agentgateway 對 MCP 2026-07-28 支援度的調研，產出 98 行研究筆記
  （`/tmp/agw-kiro-research.md`）。發現 gateway 已支援 2026-07-28 與版本交集
  行為，據此修正本 README 的適配聲明。
- **Copilot CLI**（config 移除 agentgateway 後的本 session）：同研究鏈的後續
  問題以 agwctl 完成。過程抓到兩個 UX 問題並當場修掉：`--timeout 300` 裸數字
  被拒（現在裸數字視為秒）；flag 錯誤誤回 exit 5（改回 exit 2）。
- 兩個 harness 的 `mcpServers.agentgateway` 均已移除；原設定備份在
  `~/temp/agentgateway-backups/agentgateway-20260901-233629/`。
- 當晚 ref-context upstream 500，doctor 正確回報 68/7 tools 並 exit 6。

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
