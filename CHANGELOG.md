# Changelog

All notable changes to this project are documented in this file. The format
follows Keep a Changelog; versions follow SemVer.

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