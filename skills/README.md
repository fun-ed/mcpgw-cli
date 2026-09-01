# skills/

repo 內建 agent skills。每個目錄一個 `SKILL.md`，frontmatter 帶 `name` 與 `description`。

| skill | 用途 |
|---|---|
| `agwctl-usage` | harness 以 agwctl 取代直接 MCP 連線的操作流程（find → describe → call） |
| `agwctl-maintenance` | 驗證、升級、發佈本 repo 的流程 |

## 維護規則

1. 指令、flag、exit code、預設輸出改變時，同一個 PR 內同步對應 `SKILL.md`
2. `description` 寫「什麼時候用」，不寫「這是什麼」
3. 內文以可執行指令為主；散文越少越好
4. 改完在已載入的 harness 重載（`/reload-plugins`）或新開 session
5. 接進全域 harness 用 `~/.agents/sync.sh` 的 symlink 慣例，不要手動複製
6. Review 時把 `AGENTS.md`、`skills/`、`.github/PULL_REQUEST_TEMPLATE.md`
   當同一份契約看
