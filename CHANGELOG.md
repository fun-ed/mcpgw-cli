# Changelog

All notable changes to this project are documented in this file. The format
follows Keep a Changelog; versions follow SemVer.

## [0.2.3] - 2026-11-02

### Fixed

- 成功的 `call --jq` 輸出不再被 `--max-chars` 截斷，保留完整 JSON／JSONL。
- 預設文字模式仍套用長度上限，包括工具回傳的 JSON 文字；工具錯誤仍略過 jq，
  保留原本輸出模式與退出碼。

### Changed

- MCP initialize identity 與 CLI 版本更新為 `agwctl/0.2.3`。
- 使用說明改以 jq 欄位選取或字串切片限制 jq 輸出大小。
- Patch 依賴更新：`github.com/itchyny/gojq` v0.12.19 → v0.12.20；
  保留 MCP Go SDK v1.8.0 與 Cobra v1.10.2。
- 以 mise Go 1.27.3 建置，更新專案 pin；module 最低相容版本保留 Go 1.26.7。

### Added

- CLI 回歸測試涵蓋大型 JSON 文字、完整 jq JSON、JSONL 與工具錯誤。

## [0.2.2] - 2026-10-13

### Changed

- Dependencies: `golang.org/x/sys` 0.49.0, `golang.org/x/oauth2` 0.37.0,
  `golang.org/x/sync` 0.24.0, `golang.org/x/time` 0.16.0, `spf13/pflag`
  1.0.10, `segmentio/asm` 1.2.1, `itchyny/timefmt-go` 0.1.9. `govulncheck`
  reports no vulnerabilities.
- MCP initialize identity reports `agwctl/0.2.2`.
- Releases now attach prebuilt `darwin`/`linux` `amd64`/`arm64` archives and
  `SHA256SUMS`.

## [0.2.1] - 2026-10-13

### Changed

- `github.com/modelcontextprotocol/go-sdk` 1.7.0 → 1.8.0: bounded SSE event and
  JSON nesting reads, non-blocking cancel notification, session teardown fixes.
  The negotiated protocol is unchanged (`2025-11-25`).
- MCP initialize identity reports `agwctl/0.2.1`.
- Verified against agentgateway v1.6.0 (stable), which paginates MCP list
  results; `tools list` already follows `nextCursor`, so all 76 tools on the
  8-target gateway are returned.
- Docs: to see every tool of one target, use `tools list --target <name>`;
  `tools search` returns the top 5 by default and is not a registry listing.

## [0.2.0] - 2026-09-02

### Added

- `agwctl version` and `--version`: print the client version and the repo URL.

### Changed

- `--timeout` default is now 300s (was 120s); the internal fallback timeout matches.
- MCP initialize identity reports `agwctl/0.2.0`.
- Docs: the install section notes the `~/go/bin` PATH requirement with a full-path
  fallback; the maintenance doc and the usage skill record the 2026-09-02 harness
  cutover, after which the direct `agentgateway` MCP entry is removed from every
  harness and this CLI is the only access path.

## [0.1.0] - 2026-09-01

### Added

- Gateway client: streamable HTTP with negotiated protocol version; standalone
  SSE disabled (a 12s SSE connect avoided, single run ~3s).
- `tools list [--target]`, `tools search` (lexical, top 5), `tools describe`.
- `call`: `--arg`, `--json` (`@file`), `--jq`, `--max-chars`, `--out`, and
  `--stdin` JSONL batching over one session.
- `doctor`: reachability, initialize latency, per-target tool counts,
  `--expect-targets` (exit 6 on mismatch), `--json` report.
- Tool-list cache: name, target and description only, 10 min TTL, `--refresh`.
- Exit code contract 0-6 (PLAN.md §3).
- Measured on the live gateway: `tools list` 6,653 bytes, about 91% schema-token
  saving versus a direct MCP connection.

### Fixed

- `--timeout 300` (a bare number) parses as seconds; flag errors exit 2 instead of 5.