# AGENTS.md

Instructions for AI agents working in this repository.

## Project

- mcpgw-cli is the Go repo for `agwctl`, a CLI client for the local
  agentgateway (`http://127.0.0.1:8083/mcp`, 8 targets, ~70 tools). It replaces
  direct MCP integration: harnesses resolve and call tools through the shell,
  so tool schemas never live in session context.
- Fits agentgateway v1.5.0+ legacy stateful streamable HTTP; the negotiated
  protocol version is `2025-06-18` (see README "適配範圍").
- `PLAN.md` is the single source of truth. Change it before changing behavior.

## Commands

- Build: `go build -o agwctl ./cmd/agwctl` (Go 1.26.x, pinned in .tool-versions)
- Quality gates: `go test ./...` and `go vet ./...` must be clean
- Live check: `./agwctl tools list -v` must show 8 targets / 70 tools and the
  negotiated protocol version; then one read-only `call` against deepwiki or
  duckduckgo
- The gateway itself lives outside this repo (`~/temp/agentgateway`). Never
  edit its `conf/config.yaml` or `docker-compose.yaml` from here.

## Development rules

- Develop in git worktrees: `git worktree add .worktrees/<name> -b <branch>`.
  Keep main clean; merge fast-forward; remove the worktree when merged.
- Surgical changes only. Zero comments unless explaining a why.
- Dependencies: stdlib first, pinned versions, one-line reason in PLAN.md §5.
- Output contract (PLAN.md §3): stdout carries data only, logs go to stderr;
  `--json` output stays minimal; `--max-chars` never cuts JSON; a tool
  `isError` prints its content and exits 1; `--jq` is skipped on isError.
- Exit codes: 0 ok, 1 tool error, 2 usage, 3 connect, 4 timeout, 5 protocol,
  6 doctor target mismatch.

## Skills and agent-file maintenance

- `skills/` holds agent skills (`SKILL.md` with frontmatter). They document
  behavior, so update them in the same PR as any flag, output or exit-code
  change. A behavior change that skips them is an incomplete PR.
- Keep `description` in the frontmatter as a "when to use" sentence, not a
  "what is this" sentence.
- After editing skills in a harness that already loaded them, reload
  (e.g. `/reload-plugins`) or start a new session.
- `AGENTS.md`, `skills/README.md` and `.github/PULL_REQUEST_TEMPLATE.md` are
  maintained together; review them as one unit.

## PR and commit rules

- Commits: `type: summary` with a bullet body. NO co-author trailers and NO AI
  signatures of any kind. Ever.
- PR flow, review checklist and handoff definition live in `docs/pr-handoff.md`
  and `docs/quality.md`.
- Never write secrets into source, docs, logs or commits. The gateway needs no
  auth and agwctl needs no tokens.
