# AGENTS.md

Instructions for AI agents working in this repository.

## Project

- mcpgw-cli is the Go repo for `agwctl`, a CLI client for the local
  agentgateway (`http://127.0.0.1:8083/mcp`, 8 targets / 80 tools verified on 2026-11-08; upstream counts vary). It replaces
  direct MCP integration: harnesses resolve and call tools through the shell,
  so tool schemas never live in session context.
- Fits agentgateway v1.5.0+ legacy stateful streamable HTTP; the negotiated
  protocol follows upstream negotiation (see README "適配範圍").
- `PLAN.md` is the single source of truth. Change it before changing behavior.

## Commands

- Build: `mise exec -- go build -o agwctl ./cmd/agwctl` (Go 1.27.3, pinned in .tool-versions)
- Quality gates: `mise exec -- go test ./...` and `mise exec -- go vet ./...` must be clean
- Live check: `./agwctl doctor --expect-targets 8` and `./agwctl tools list -v`
  must agree with the live gateway; run an actual read-only search and inspect
  result URLs. `doctor` and exit 0 alone do not prove useful tool results.
- `./agwctl help mcp` must work offline. Help changes require the offline-help
  tests and matching README/skills guidance; keep transport and exit codes stable.
- The gateway itself lives outside this repo (`~/temp/agentgateway`). Never
  edit its `conf/config.yaml` or `docker-compose.yaml` from here.

## Development rules

- Develop in git worktrees: `git worktree add .worktrees/<name> -b <branch>`.
  Keep main clean; merge fast-forward; remove the worktree when merged.
- Surgical changes only. Zero comments unless explaining a why.
- Dependencies: stdlib first, pinned versions, one-line reason in PLAN.md §5.
- Output contract (PLAN.md §3): stdout carries data only, logs go to stderr;
  `--json` output stays minimal; successful `--jq` output is not truncated.
  Default text output observes `--max-chars`; tool `isError` prints content
  and exits 1; `--jq` is skipped on isError.
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
