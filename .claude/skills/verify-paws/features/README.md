# paws feature map

One file per user-facing feature. Each names the cases that prove it.

| Feature | File | Cases |
|---|---|---|
| require-skill gate | [require-skill.md](require-skill.md) | `require-skill-commit`, `catalog-humanizer-commit` |
| block, warn, audit | [policy-actions.md](policy-actions.md) | `block-tag`, `audit-status` |
| exec-script | [exec-script.md](exec-script.md) | `exec-script-commit` |
| prompt-inject | [prompt-inject.md](prompt-inject.md) | `prompt-inject` |
| session-start | [session-start.md](session-start.md) | `session-start-require-skill` |
| advise | [advise.md](advise.md) | `advise-once` |
| MCP and file matchers | [matchers.md](matchers.md) | `mcp-send-require-skill`, `file-glob-require-skill` |
| when: conditions | [when.md](when.md) | `when-dependency-hit`, `when-dependency-miss`, `when-no-edit-tdd` |
| on_miss digest | [digest.md](digest.md) | `digest-tag` |
| packs and skill install | [packs.md](packs.md) | `launch-journey` |
