# 維護準則

## 例行檢查

```bash
./agwctl tools list --json | jq 'group_by(.target) | map({(.[0].target): length}) | add'
./agwctl tools list -v          # stderr 顯示協商的 protocol version
# 對照 gateway 端：
cd ~/temp/agentgateway && ./agw-mcp.py verify
```

- tool-list cache 在 `os.UserCacheDir()/agwctl/`，可以整個刪掉，下次自動重建
- cache 只含 name/target/description，schema 永遠活取，不會有 schema 腐爛問題
- harness 端自 2026-09-02 cutover 起不再掛 `agentgateway` MCP；唯一存取路徑是
  本 CLI。舊 session 的 `mcp__agentgateway__*` 是遺留，不要依賴
- `agwctl` 裝在 `/usr/local/bin`（symlink 指向 `~/go/bin/agwctl`，2026-09-02
  建立），全部 shell 含 GUI 啟動的 harness 都找得到；rebuild 只寫
  `~/go/bin/agwctl`，symlink 自動跟上
- bare `agwctl` 出現 `command not found`：先查 `/usr/local/bin/agwctl` symlink
  還在不在，斷了就重建 `sudo ln -s "$HOME/go/bin/agwctl" /usr/local/bin/agwctl`

## agentgateway 升級時

1. 先升 gateway 本體（runbook 在 `~/temp/agentgateway/README.md`）
2. 本 repo 重跑全套驗收：`tools list`、`tools describe`、一個唯讀 `call`、
   一個 `--stdin` batch
3. 看 `tools list -v` 的協商版本有沒有變
4. gateway 支援 MCP 2026-07-28（stateless、`server/discover`）時：
   評估 go-sdk 版本 → 更新 `PLAN.md` 適配聲明與風險表 → 全綠才改 README
5. go-sdk 跟著 MCP spec release 走，不要為了「看起來新」硬升

## 依賴升級

- patch 級（x.y.Z）直接升，跑測試即可
- minor 級（x.Y.0）升完在 `PLAN.md` §5 更新版本表
- go-sdk 升級後必須重驗 protocol 協商行為（M1 的驗收方式）

## Release

```bash
git tag vX.Y.Z && git push origin vX.Y.Z
```

- 版本語意跟著 binary 功能面走；PLAN.md 狀態表同步
- 私有 repo，不安裝到 brew；安裝就是 `go build`

## Skills / Agents 檔案的維護

這些檔案是「行為文件」，不是裝飾：

| 檔案 | 內容 |
|---|---|
| `AGENTS.md` | AI agent 在本 repo 的操作守則 |
| `skills/agwctl-usage/SKILL.md` | harness 用 agwctl 的操作流程 |
| `skills/agwctl-maintenance/SKILL.md` | 驗證、升級、發佈流程 |
| `skills/README.md` | skill 維護規則 |
| `.github/PULL_REQUEST_TEMPLATE.md` | PR 模板 |

規則：

1. 指令、flag、exit code、預設輸出改變時，同一個 PR 內同步對應檔案
2. SKILL.md frontmatter 的 `description` 寫「什麼時候用」，不寫「這是什麼」
3. 內文以可執行指令為主，散文越少越好
4. 改完 skill 在已載入的 harness 重載（`/reload-plugins`）或新開 session
5. 要接進全域 harness 時走 `~/.agents/sync.sh` 的 symlink 慣例，不手動複製
