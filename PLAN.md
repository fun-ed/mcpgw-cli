# agwctl 設計與實做計畫（SOT）

| | |
|---|---|
| 狀態 | v0.1 定稿（含 M0 實測與 Opus-5 judge 審查），未開工 |
| 日期 | 2026-09-01 |
| 語言 | Go 1.26.x（目前最新 patch go1.26.7，2026-08-19） |
| 位置 | `~/temp/agentgateway/agwctl/`，遠端 `github.com/fun-ed/mcpgw-cli`（私有） |
| 對象 | `http://127.0.0.1:8083/mcp`，8 targets，約 70 tools |

本文件是 agwctl 的 single source of truth。實做時以此為準；行為有變先改這裡。

## 1. 為什麼要這個

現況：每個 harness（Claude Code、Copilot CLI、Kiro、Codex、Devin）透過 agentgateway 連上後，`tools/list` 的完整 schema 會全部放進 session context。8 個 target、約 70 個 tool，每個 session 每個 turn 都重複背這份 schema。

Uber 的模式把這層換掉：「CLI tool resolution replaces direct MCP integration」。Harness 只知道一個 shell 指令，真正要呼叫時才去 gateway resolve tool，schema 不存在 session context 裡。實測數字（Uber blog "Running a Software Factory Efficiently at Uber Scale"，以及 philschmid.de 的 mcp-cli）：六個 MCP server 的 schema 從約 47k tokens 降到約 400 tokens，同樣任務 MCP 比 CLI 貴 4 到 32 倍。

agwctl 直接省前兩層，第三層靠它保持 script-friendly（見 §4）：

1. Schema 層。`tools list` 只回名稱加一行描述，schema 要 `tools describe` 才印。
2. Result 層。`call` 支援 `--jq` 和 `--max-chars`，500 KB 的回應不會整坨塞回 LLM。
3. Workflow 層（Code Mode）。多步流程（query → poll → fetch → 總結）不讓 LLM 每步參與：agent 寫一段 shell script，在一個 Bash tool call 內跑完輪詢與過濾，只有最終結果回到 LLM。形狀從 `LLM ↔ tool ↔ LLM ↔ tool` 變成 `LLM ↔ deterministic program ↔ LLM`。這層不是 agwctl 的產品能力，它的責任只有輸出契約、batch 模式和 SKILL.md 教學；不做 script runner，Bash 就是 orchestrator。

## 2. 架構

```text
Copilot CLI / Claude Code / Kiro / Codex / Devin
        │
        │ Bash
        ▼
      agwctl            ← 本專案，單一 static binary
        │ initialize / tools/list / tools/call（streamable HTTP, stateful）
        ▼
   agentgateway v1.5.0  ← 已存在，不改
        │
        ├─ Hindsight    ├─ Exa    ├─ OpenViking
        ├─ DeepWiki     ├─ Tavily ├─ Ref Context
        └─ brave-search └─ duckduckgo-search
```

一次呼叫的流程：

```text
1. agwctl tools search "search the web"   →  top 5 名稱（~100 tokens）
2. agwctl tools describe brave-search_web_search
                                          →  只印這個 tool 的 inputSchema
3. agwctl call brave-search_web_search --arg query="..." --jq '.content[0].text'
                                          →  只印結果（可截斷）
```

單次呼叫是一個新 process：initialize → 動作 → DELETE；batch 模式同一個 process 只 initialize 一次（見 §4）。跟今天 `agw-mcp.py` 的行為一致，gateway 端不用任何改動。

## 3. v0.1 指令規格

```text
agwctl tools list                      全部 tool：名稱 + 一行描述（不帶 schema）
agwctl tools list --target deepwiki    只列某 target
agwctl tools search QUERY [--limit N]  lexical 排序，預設 top 5
agwctl tools describe NAME             印 name、description、inputSchema
agwctl call NAME [--arg k=v]... [--json '{...}'|@file.json]
                   [--jq EXPR] [--max-chars N] [--out FILE]
agwctl call --stdin                    batch：stdin 每行一個 {"name","arguments"}，
                                       每行回一個 result，整個 batch 只 initialize 一次
agwctl doctor [--expect-targets N]     gateway 可達性、initialize 延遲、
                                       每 target tool 數、異常提示；永遠活取，
                                       不吃 tool-list cache
```

### 輸出契約（最重要的一條）

- stdout 只放資料，stderr 只放 log 和警告。pipe 和 redirect 才會乾淨。
- 錯誤分兩種，走不同出口：tool 自己回的 `isError` 是「工具的資料」，照印 stdout、exit 1，agent 要讀它自我修正；agwctl 層的 transport / protocol / gateway 錯誤走 stderr 一行 `agwctl: <kind>: <message>`、stdout 不出東西，exit code 分支（3/4/5）。不做 JSON error envelope，exit code 就是 script 的分支訊號。
- 預設文字輸出給 agent 讀，格式用 tab 分隔欄位，不用表格框線（padding 浪費 token）。
- `--json` 輸出最小 JSON：`tools list` 給 `{"name","target","description"}`，`search` 加 `{"score"}`。絕不順手帶 inputSchema、annotations、_meta。
- `call --json` 是例外：印完整 MCP `CallToolResult` 的 compact JSON。這是給 script 串接用的原始形狀，`--jq` 就作用在這份 JSON 上。
- 處理順序固定：`raw CallToolResult → --jq → --max-chars → --out/stdout`。`--jq` 隱含 JSON 語意；`isError` 時 `--jq` 不執行，原樣輸出錯誤，避免 `.content[0].text` 靜默取到錯誤字串。
- `--max-chars` 預設 20000，超過就截斷，結尾補一行截斷說明。`0` 表示不限制。**不作用在 `--json` 上**（截 compact JSON 會出非法 JSON）；`--json` 要限量用 `--jq` 或 `--out` 落盤。
- `--out FILE` 把完整結果寫進檔案，stdout 只印路徑和 byte 數：有 `--jq` 時寫 post-jq 內容，沒有時寫 raw CallToolResult JSON，下游 jq 一定吃得動。
- `call --jq .` 就是 raw result JSON 模式，腳本串接用這個。
- `call --stdin` batch 輸出 JSONL，每行一個 CallToolResult；batch 忽略 `--max-chars` 與 `--jq` 並往 stderr 警告（腳本自己 pipe jq）。transport 錯誤中止 batch、exit 5；單行 isError 不中止，全部跑完後 exit 1。

### Exit codes

| code | 意義 |
|---|---|
| 0 | 成功（search 沒命中也是 0，空輸出是合法答案） |
| 1 | tool 回 `isError`；錯誤內容照印 stdout，agent 要讀它自我修正 |
| 2 | 參數用法錯誤 |
| 3 | gateway 不可達或 initialize 失敗 |
| 4 | 逾時（`--timeout`，預設 120s；doctor 對 up-check 用 10s） |
| 5 | 執行中的 transport / protocol 錯誤（非 tool 錯誤），script 該重試或回報 |
| 6 | `doctor --expect-targets` 檢查不符 |

### `call` 參數規則

- `--arg k=v` 可重複。值先試 JSON 解析：`5` 是數字、`true` 是布林、`"x"` 是字串，其餘當純字串。
- `--json` 接完整 JSON 物件，或 `@path/to/file.json` 讀檔。與 `--arg` 互斥。
- `--jq EXPR` 跑在整個 MCP `result` 物件上：有 `structuredContent` 用 `.structuredContent...`，純文字結果用 `.content[0].text`。規則單一，不猜。
- 兩種都不給就是 `{}`（很多 tool 沒參數）。

### 不做的（YAGNI，明確列出）

不搞 embedding、vector DB、SQLite、daemon、plugin system、schema 本地快取、resource/prompts 支援、auth（gateway 本機無 auth）、config file（URL 固定，`--url` 和 `AGWCTL_URL` env 就夠；有需要 v0.2 再說）。工具清單只快取名稱與一行描述、短 TTL 加 `--refresh`（見 §4），schema 永遠不快取。

## 4. 關鍵設計決策

### MCP client：用官方 SDK，備妥 fallback

用 `github.com/modelcontextprotocol/go-sdk` v1.7.0。理由：MCP spec 2026-07-28 大改（stateless core、`server/discover` 取代 initialize），自己追 wire format 不划算。v1.7.0 保留對 2025-11-25 以前的相容，連線時協商雙方都支援的最高版本。

agentgateway v1.5.0 對 mixed upstream 走 legacy stateful 流程（initialize handshake + `mcp-session-id`）。實測協商在 `2025-06-18` 與 `2025-11-25` 之間浮動（皆 legacy stateful，相容）。cutover 調研確認 gateway 本體已支援 MCP 2026-07-28 的 stateless `server/discover`，但 multiplexing 協商取全部 upstream 的版本交集，任何一個 legacy upstream 都把整體壓在 legacy；agwctl 跟著協商結果走，不單方面升級。

Fallback 路徑已驗證可行：`agw-mcp.py` 證明這個 gateway 只需要 `POST /mcp`、`Accept: application/json, text/event-stream`、`mcp-session-id` header、SSE `data:` 行解析，手寫 thin JSON-RPC client 約 150 行。SDK 若在 M1 撞牆，降級走這條，不影響其他設計。

### Session 與快取

SDK 的 `StreamableClientTransport` 自己管理 session id，沒有公開的「帶入舊 session id」入口，跨 process session 快取做不乾淨。v0.1 的對策有兩個，都已足夠：

- Batch 模式。`call --stdin` 在一個 process 內用同一個 session 跑 N 個請求，只 initialize 一次。Code Mode 迴圈要省 initialize 成本就改成一行一個 request 餵進 `--stdin`，控制流仍在 Bash。SDK 在單一 process 內跑多個 call 完全沒問題，這是「不換掉 SDK」的前提。
- 工具清單快取。只快取 `name`、`target`、`description`（不含 schema），放 `os.UserCacheDir`，TTL 10 分鐘，`--refresh` 強制重取，`doctor` 永遠活取。`search`、`describe` 命中快取時不打 gateway；upstream 改名靠短 TTL 自癒，schema 永遠不落地所以不會腐爛。

實測：sidecar 溫機後 initialize 0.48s（2026-09-01，對活 gateway）。單次呼叫付 0.5s 可接受；`doctor` 持續量測並回報，若劣化再評估 thin client 與跨 process session 重用（v0.2）。DELETE 照 README 規則送（batch 送一次）。

M1 追加實測：go-sdk 預設在 initialize 後開一條 standalone SSE GET stream，對這個 gateway 要 12s 才完成連線；CLI 不需要 server-initiated 訊息，設 `DisableStandaloneSSE: true` 後單次執行降到約 3s。gateway 的每請求 fanout floor 約 0.5s（raw 協議四步：initialize 0.5s、initialized 0.5s、tools/list 0.6s、DELETE 0.4s）。

### 搜尋：tokenized lexical scoring，約 60 行，零依賴

```text
score = Σ(名稱命中 3.0 ＋ 名稱前綴命中 2.0 ＋ target 名命中 1.5 ＋ 描述命中 1.0)
```

query 先 lowercase、切非英數 token，stopword 過濾並跳過長度小於 3 的 token（`"search the web"` 的 `the` 會命中全部），每個欄位的命中最多計一次，避免長描述的 tool 系統性壓過名稱精準命中。幾百到一千個 tool 內這樣就夠，不做 BM25 的 IDF，不做 fuzzy typo 容錯（真需要再考慮 `sahilm/fuzzy`，零風險的小依賴）。排序：score 降冪，同分用名稱字母序，輸出取 top `--limit`（預設 5）。

### 設定優先序

`--url` flag > `AGWCTL_URL` env > 預設 `http://127.0.0.1:8083/mcp`。不需要 token。不讀 `conf/config.yaml`（裡面有 secrets 展開的引用，也不該由 client tool 碰）。

### Script-friendly 是 Code Mode 的基礎

Tool search 只回答「用哪個 tool」，不回答「這個 workflow 該不該讓 LLM 每步都參與」。輪詢、分頁、對多結果逐筆處理這類流程，正確做法是 agent 產生一段 script，在一個 Bash tool call 內跑完：

```bash
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
```

兩條配套：MCP tool 幾乎不宣告 outputSchema，SKILL.md 要教「先做一次探測 call 看輸出形狀，再寫 jq」；可執行範本放 `examples/`，M2/M3 對唯讀 target 實跑過。這對 agwctl 的硬性要求：JSON 進 JSON 出、exit code 穩定、stdout 純淨，就是 §3 的輸出契約，不另立功能。`agwctl flow`、script runner、workflow 引擎全部 YAGNI，Bash 加 jq 就是那個 deterministic program。

## 5. Module 與依賴（2026-09 實查版本）

查詢時間 2026-09-01，來源 proxy.golang.org：

| 依賴 | 版本 | 用途 |
|---|---|---|
| Go toolchain | go1.26.7 | 語言：`new(expr)`、Green Tea GC、`errors.AsType` |
| `github.com/modelcontextprotocol/go-sdk` | v1.7.0 | MCP client（streamable HTTP） |
| `github.com/spf13/cobra` | v1.10.2 | CLI 子指令、flag |
| `github.com/itchyny/gojq` | v0.12.19 | `--jq` 的純 Go jq engine |

其餘全用 stdlib：`net/http`（SDK 內部也用它）、`encoding/json`、`log/slog`（stderr）、`os.UserCacheDir`。不做表格、不做顏色、不做 fuzzy 套件。

```text
agwctl/
  go.mod                 module github.com/fun-ed/mcpgw-cli / go 1.26（binary 仍叫 agwctl）
  cmd/agwctl/main.go
  internal/cli/          cobra 指令定義（tools, call, doctor）
  internal/gw/           SDK 包一層：connect, listTools, callTool, close
  internal/search/       評分器（純函數，最好測）
  internal/out/          文字/JSON 輸出、jq、截斷
```

## 6. 程式碼骨架

```go
// internal/gw/client.go
func Connect(ctx context.Context, url string, timeout time.Duration) (*Client, error) {
    impl := &mcp.Implementation{Name: "agwctl", Version: "0.1.0"}
    c := mcp.NewClient(impl, nil)
    tr := &mcp.StreamableClientTransport{Endpoint: url}
    sc, err := c.Connect(ctx, tr) // SDK 內部做 initialize、協商版本
    ...
}

// tools/call 後抽文字：result.Content 的 TextContent 串接輸出；
// isError 時照印內容、exit 1。
```

```go
// internal/search/score.go
type Tool struct{ Name, Target, Description string }
func Score(t Tool, tokens []string) float64
func Search(tools []Tool, query string, limit int) []Ranked
```

## 7. Token 量測（M0 已完成，2026-09-01 對活 gateway 實測）

Baseline 已量，閘門已過。這些是 M1 之後所有量測的對照組：

| 量項 | 實測 | 說明 |
|---|---|---|
| 溫機 initialize 延遲 | 0.48s | 對活 gateway，單次 |
| 改善前：70 tools 完整 schema | 40,740 chars ≈ 10.2k tokens | 每個 session 每個 turn 都背 |
| 改善前：70 tools 描述 | 33,826 chars ≈ 8.5k tokens | 同上 |
| 改善前合計 | ≈ 18.6k tokens / session | 本專案的目標數字 |
| cutover 後 session schema | 0（兩個 harness 已移除 `mcpServers.agentgateway`） | 2026-09-01 生效 |
| `tools list` 無 schema 版 | 6,653 bytes ≈ 1.7k tokens（一次性） | 約省 91%（M1 實測，與預估 6,579 一致） |
| 單次執行延遲 | 約 3.0s（initialize 0.5s、fanout floor 每請求約 0.5s） | batch 模式（M2）消除多步重複成本 |
| `tools search` top 5 | 預估 < 150 tokens | 實做後複測 |
| `call --max-chars` 截斷 | 對大回應 target 測試 | 實做後驗證截斷標記 |

## 8. Milestones 與驗收

| 階段 | 內容 | 驗收條件 |
|---|---|---|
| M0 | baseline 量測 | 已完成（§7），優先序已驗證 |
| M1 | module 骨架、gw client、`tools list`（含 `--json`、`--target`） | **完成 2026-09-01**：8 targets / 70 tools 與 `agw-mcp.py verify` 一致；協商結果 `2025-06-18`；`go vet`、`go test ./...` 乾淨；開發在 `.worktrees/` worktree 進行 |
| M2 | `search`、`describe`、`call` 全套 flag（含 `--stdin` batch） | **完成 2026-09-01**：對 deepwiki、duckduckgo 實際 call 成功（含 `--jq '.content[0].text'`）；截斷標記運作；batch 兩請求一 session JSONL 輸出；`--out` 輸出 jq-able JSON；search 命中快取 0.024s；scoring／cache／out 單元測試齊 |
| M3 | `doctor`、安裝文件、真實 cutover | **doctor 完成 2026-09-01**：全綠 exit 0；`--expect-targets` 不符 exit 6；`--json` 結構化報告；活 gateway 實測當晚 ref-context upstream 500，doctor 正確回報 7/8 targets、68 tools 並 exit 6（agw-mcp.py verify 同步證實是 upstream 問題）。協商版本在 `2025-06-18` 與 `2025-11-25` 之間浮動（皆為 legacy stateful，相容）。剩餘：gateway README 指引小節、真實 cutover |

完成定義：不改 `conf/config.yaml`、不動 docker-compose；gateway 的 `verify` 與 `smoke` 行為不受影響；macOS arm64 單一 binary；`go build -o ~/go/bin/agwctl ./cmd/agwctl` 可安裝。

## 9. 風險與對策

| 風險 | 對策 |
|---|---|
| go-sdk v1.7.0 對 legacy stateful gateway 的協商行為不如預期 | M1 第一天實測；fallback 是已驗證的 thin JSON-RPC client（agw-mcp.py 同款 wire format） |
| 每次執行都要 initialize，冷 sidecar 時可能拖到數秒 | 溫機實測 0.48s（§7）；`call --stdin` batch 讓多步流程只付一次；doctor 量測回報，劣化再走 v0.2 thin client |
| 工具回應過大塞爆 context | `--max-chars` 預設截斷 + `--jq`；SKILL.md 指導 agent 用 `--out` 落盤 |
| gojq 增加體積（單 binary 約多 1-2 MB） | 換來完整 jq 語意，值得；真的不要就退回 `--jq` 只支援簡單路徑 |
| upstream 改 tool 名（DeepWiki 前例） | 名稱/描述快取只留 10 分鐘 TTL 且可 `--refresh`；schema 永遠活取 |
| failOpen 讓掛掉的 target 靜默少一批 tool，agent 會誤以為 tool 不存在而自己編替代方案 | `tools list`／`search` 在 target 數少於 `--expect-targets` 時往 stderr 警告；`doctor --expect-targets 8` 用 exit 6 反映 |

## 10. 審查紀錄

**ChatGPT 建議（2026-09-01）**：採納 thin v1、`tools list` 不帶 schema、describe 才印 schema、search top-5 lexical、極小 `--json`、`--arg`/`--json @file`/`--jq`、`doctor`、cobra + 官方 SDK 優先、YAGNI 清單。調整兩處：`config.yaml` 延後（本機 gateway 無 auth、URL 固定，env + flag 覆蓋需求）；獨立 `servers` 子指令移除（由 `doctor` 和 `tools list --target` 覆蓋）。

**Opus-5 judge 審查（2026-09-01，Kiro dispatch，persona-text profile）**：判決 adopt with changes，12 項發現採納 11 項——batch 模式（`call --stdin`）、M0 baseline 閘門（已完成實測）、M3 真實 cutover 與 shell 允許清單、`--max-chars` 不作用於 `--json`、exit 5 拆分、錯誤出口二分與 `--jq` isError 行為、pipeline 順序明文化、名稱/描述短 TTL 快取、failOpen 降級警告與 `doctor --expect-targets`、範例 script 修正與 examples/ 實跑、scoring stopword 與命中上限。未採納 1 項：「thin client 升為 primary」——judge 的依據是 batch 會被 SDK 擋住，但 batch 在單一 process 內跑，SDK 完全支援；跨 process session 重用才需要 thin client，維持 v0.2 條件式觸發。