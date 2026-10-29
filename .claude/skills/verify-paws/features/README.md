# paws feature map

One file per user-facing feature. Each names the cases that prove it.

| Feature | File | Cases |
|---|---|---|
| require-skill gate | [require-skill.md](require-skill.md) | `require-skill-commit`, `catalog-humanizer-commit`, `compaction-reset`, `resume-no-redeny`, interactive `require-skill-tui` |
| session-start | [session-start.md](session-start.md) | `session-start-require-skill` |
| MCP and file matchers | [matchers.md](matchers.md) | `mcp-send-require-skill`, `file-glob-require-skill` |
| when: conditions | [when.md](when.md) | `when-dependency-hit`, `when-dependency-miss`, `when-no-edit-tdd`, `when-repo-marker-hit`, `when-repo-marker-miss`, `when-new-file`, `when-agent-count` |
| on_miss digest | [digest.md](digest.md) | `digest-tag`, `aws-s3-digest`, `secret-scan-commit` |
| packs and skill install | [packs.md](packs.md) | `launch-journey`, `practice-testing-pack` |
| local telemetry | [telemetry.md](telemetry.md) | `require-skill-commit` |
| paws skill | [paws-skill.md](paws-skill.md) | `paws-skill-writes-rule`, `paws-skill-*` (25 more) |
