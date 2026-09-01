---
name: agwctl-maintenance
description: Verify, upgrade or release the mcpgw-cli repo (agwctl). Use when asked to check gateway tool health, bump dependencies, re-verify after an agentgateway upgrade, or tag a release.
---

# agwctl maintenance

## Quality gates

```bash
cd <repo-root-or-worktree>
go build -o agwctl ./cmd/agwctl && go vet ./... && go test ./...
```

All three must pass before anything else. Go toolchain is pinned in
`.tool-versions` (1.26.7, via mise).

## Live checks against the gateway

```bash
./agwctl tools list --json | jq 'group_by(.target) | map({(.[0].target): length}) | add'
./agwctl tools list -v                  # negotiated protocol version on stderr
./agwctl tools search "search"          # should be fast after the first run (cache)
./agwctl call deepwiki_read_wiki_structure --arg repoName=agentgateway/agentgateway --max-chars 300
```

Compare target and tool counts with `~/temp/agentgateway/agw-mcp.py verify`
(8 targets / 70 tools as of 2026-09-01; remote capability may change).

## After an agentgateway upgrade

1. Upgrade the gateway first (runbook in `~/temp/agentgateway/README.md`)
2. Re-run the live checks above; watch the negotiated protocol version
3. If the gateway gains MCP 2026-07-28 (stateless, `server/discover`),
   evaluate a go-sdk bump and update `PLAN.md` fit statement and risk table
4. Only after everything is green, update the README fit line

## Dependencies

- go-sdk follows MCP spec releases; cobra and gojq are boring pins
- patch bumps: just run the gates; minor bumps: update the table in `PLAN.md` §5
- add a one-line reason to `PLAN.md` §5 for any new dependency

## Release and docs sync

- `git tag vX.Y.Z && git push origin vX.Y.Z`; update the status table in `PLAN.md`
- Behavior changes (flags, output, exit codes) require same-PR updates to
  `AGENTS.md`, `skills/README.md` and `skills/*/SKILL.md`
- The tool-list cache lives in `os.UserCacheDir()/agwctl/`; deleting it is safe
