# agwctl

CLI client for the local agentgateway (`http://127.0.0.1:8083/mcp`).
Shell-based tool resolution replaces direct MCP integration, so harnesses
hold one command instead of every tool schema.

Design and scope live in [`PLAN.md`](PLAN.md). Development happens in git
worktrees under `.worktrees/`, never on a bare checkout of `main`.