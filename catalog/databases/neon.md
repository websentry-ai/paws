---
name: neon
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?|bunx\s+|pnpm\s+(dlx|exec)\s+|yarn\s+(dlx\s+)?)?(neonctl|neon)(@\S*)?\s+(init|env|deploy|checkout|auth|link|logs|login|config|bootstrap|status|profile|pins|postgres|pg|operations|mcp|diff|inspect|functions|branches?|projects?|orgs|me|connection-string|cs|databases?|roles?|computes?|set-context|lakebase|dev|sql|psql)\b
action:
  type: require-skill
  skill: neon
upstream:
  repo: github.com/neondatabase/agent-skills
  path: skills/neon/SKILL.md
  ref: 392174f40372f8ffd1f02604cbd68310c47fc7ac
  content_hash: sha256:c2cc99203a8569b068e7503e956138600a61bbbb772d43c795ab1382cc61723d
provenance:
  stars: 112
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
tags:
- practice
- databases
---
Loads neon before the Neon CLI or a Neon MCP tool is used, so the branch-first workflow and primitives guidance apply.
