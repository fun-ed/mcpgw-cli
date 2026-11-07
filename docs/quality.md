# Review / Eval / Test 準則

## Test

- `go test ./...` 必綠、`go vet ./...` 必乾淨，這是每個 PR 的最低門檻
- 現有單元測試分佈：
  - `internal/search`：tokenize、stopword、field cap、tie-break、limit
  - `internal/out`：截斷標記、jq 求值
  - `internal/gw`：in-memory transport、cache TTL / URL / corrupt、connect sentinel
  - `internal/cli`：args 解析、usage 錯誤
- 新行為 = 新測試；修 bug = 先寫紅測試再修
- 不引入重型測試框架；gateway 互動用 in-memory transport 或真 gateway

## Eval（對活 gateway 的驗收清單）

碰到 `internal/gw/` 或 `internal/cli/` 的 PR，merge 前跑：

```bash
./agwctl help mcp                        # 離線 help 測試不應發送網路請求
./agwctl doctor --expect-targets 8       # registry 檢查，不代表 tool call 成功
./agwctl tools list                      # 與即時 agw-mcp.py verify 一致，數量隨 upstream 改變
./agwctl tools list -v                   # 協商 protocol version 沒有意外
./agwctl tools search "search"           # 命中合理，快取後 < 0.1s
./agwctl tools describe duckduckgo-search_search
./agwctl call duckduckgo-search_search --arg query="model context protocol" --arg max_results=2 --jq .
printf '%s\n' '{"name":"deepwiki_read_wiki_structure","arguments":{"repoName":"agentgateway/agentgateway"}}' \
  | ./agwctl call --stdin --max-chars 0
```

搜尋驗收需看到實際結果 URL；`isError=false`、exit 0 或 doctor 成功都不足以證明
搜尋可用。非同步寫入需保留回傳的 operation ID、等待 completed 並 read-back。

輸出大小或延遲有變時，量測寫回 `PLAN.md` §7。

## Eval 紅線

- `tools list` 預設輸出 > 5 KB、`tools search` > 500 bytes：要有 `PLAN.md` 記錄
- 單次執行 > 5s：`PLAN.md` 記錄並評估（batch、cache、thin client）
- `--json` 出現 `inputSchema`、`annotations`、`_meta`：直接打回

## Review 清單

1. stdout 純淨：資料才進 stdout，log 一律 stderr
2. exit code 契約沒被打破（0/1/2/3/4/5/6）
3. `--json` 仍是最小形狀
4. 成功的 `--jq` JSON／JSONL 輸出不截斷；預設文字模式仍有上限；`--out` 寫的東西下游 jq 吃得動
5. 快取仍只含 name/target/description
6. `PLAN.md` 先改、code 後改
7. `skills/` 與 `AGENTS.md` 同步（行為變更時）
8. commit 無 co-author、無 AI 署名
9. 任何預設輸出變大，PR 描述裡要說明 token 成本理由
