# 開發準則

## SOT

`PLAN.md` 是 single source of truth。行為、契約、量測、里程碑：先改
`PLAN.md`，再動 code。兩邊不一致時以 `PLAN.md` 為準，直到修好為止。

## 環境

- Go 1.26.x，`.tool-versions` 釘 `1.26.7`（mise 管理）
- build：`go build -o agwctl ./cmd/agwctl`
- 本機 gateway：`http://127.0.0.1:8083/mcp`，屬於 `~/temp/agentgateway`
  workspace，不在本 repo；本 repo 不碰它的任何設定

## Worktree

開發一律用 worktree，main 只收乾淨的結果：

```bash
git worktree add .worktrees/<name> -b <branch>
# ...開發、commit...
git checkout main && git merge --no-edit <branch>
git worktree remove .worktrees/<name>
```

merge 用 fast-forward 優先；main 任何時刻都要是 green（vet + test + live eval）。

## 依賴

- stdlib 優先：`net/http`、`encoding/json`、`log/slog`、`os.UserCacheDir`
- 新依賴要 pinned version，並在 `PLAN.md` §5 加一行理由
- `go.mod` 與 `PLAN.md` §5 的版本表必須一致
- 現有依賴：`modelcontextprotocol/go-sdk`、`spf13/cobra`、`itchyny/gojq`

## Code style

- 零註解預設；只寫「為什麼」（隱藏限制、workaround、取捨）
- 命名沿用專案既有詞彙：`target`、`ToolRow`、`result`、`row`
- surgical changes：只動請求相關的行，不順手重構、不順手美化
- 契約在 `PLAN.md` §3：stdout 純淨、exit code、最小 JSON、截斷規則

## Secrets

- `.env`、token、Authorization header 不進 source、docs、log、commit
- gateway 本機無 auth；`agwctl` 不需要任何 secret，也不要替它加

## Skills / agent 檔案

指令、flag、exit code、預設輸出改變時，同一個 PR 同步
`skills/*/SKILL.md`、`AGENTS.md`、`skills/README.md`。維護規則見
`docs/maintenance.md`。
