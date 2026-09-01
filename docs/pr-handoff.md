# 交接 PR 準則

## Branch 與 commit

- branch 在 `.worktrees/` 內開，名稱 `<topic>-<scope>`（例：`m3-doctor`）
- commit 格式：`type: summary`（feat / fix / docs / chore / refactor），body 列要點
- **一律不可 co-author，一律不可 AI 署名**（含任何形式的 trailer）
- 個人 repo，不需要 Jira ticket；需要追溯時 PR 描述寫清楚動機

## PR 流程

1. worktree 內完成開發與測試
2. push branch：`git push -u origin <branch>`
3. 開 PR，模板自動帶入 `.github/PULL_REQUEST_TEMPLATE.md`
4. self-review：照 `docs/quality.md` 的 Review 清單逐條過
5. merge：fast-forward 優先；main 必須保持 green
6. merge 後 `git worktree remove .worktrees/<name>`

## PR 描述要求

- 動機指向 `PLAN.md` 的條目或里程碑
- 驗收證據貼指令與輸出，不貼截圖
- 行為變更必須列出 `PLAN.md` / `AGENTS.md` / `skills/` 的 diff 位置
- 預設輸出變大時說明 token 成本影響

## 交接給另一個 agent 或人

接手者只需要四樣東西：

1. `PLAN.md` — 現況、契約、量測、里程碑
2. `docs/development.md` — 怎麼開發（worktree、依賴、style）
3. `docs/quality.md` 的 Eval 清單 — 怎麼驗
4. `git log` — 已經做了什麼

交接完成的定義：接手者能跑通 Eval 清單，並能解釋每個 exit code 的語意。
