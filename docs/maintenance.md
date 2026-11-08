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
- 直接 MCP 與 agwctl 可並存；以目前 harness 設定與生成的 inventory 為準。
  需要避免 schema 常駐 context 時優先使用 CLI，工具名稱仍須從即時 registry 取得。
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

1. 更新 `internal/gw/client.go` 的 `ClientVersion`、`PLAN.md` 狀態與 `CHANGELOG.md`。
2. 執行 `mise exec -- go test ./...`、`mise exec -- go vet ./...`，核對 binary 的
   `version` 與 `--version`，再 commit 並推送。版本常數也用於 MCP initialize identity。
3. 從乾淨的已提交 revision，以 `CGO_ENABLED=0` 與 `mise exec -- go build -trimpath`
   建置 darwin/linux × amd64/arm64。每個 tar.gz 使用
   `agwctl_<version>_<os>_<arch>.tar.gz`，附 `SHA256SUMS`；產物放 repo 外。
4. 建立指向已驗證 commit 的 tag，再推送該 tag。以 CHANGELOG 與實際驗證結果
   撰寫 release notes，不把未測平台寫成執行驗證通過。

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
gh release create vX.Y.Z --repo fun-ed/mcpgw-cli --verify-tag \
  --title "vX.Y.Z" --notes-file /path/to/release-notes.md \
  /path/to/artifacts/*.tar.gz /path/to/artifacts/SHA256SUMS
```

5. 重新下載 release assets，核對 SHA256SUMS、archive 結構與平台格式；本機平台
   解壓執行 `version`、`help mcp` 與 gateway doctor。交叉編譯不代表其他平台已實跑。
6. 本機安裝獨立驗證：`mise exec -- go build -o "$HOME/go/bin/agwctl" ./cmd/agwctl`，
   再用 PATH 上的 `agwctl version` 確認。私有 repo 不發佈至 Homebrew。

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
