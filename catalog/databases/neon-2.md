---
name: neon-2
event: pre-tool
matcher:
  tool: mcp__*neon*__*|mcp__*Neon*__*
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
summary: Sets up Neon Postgres, auth, storage, functions and AI gateway via the neon CLI or MCP, branch-first.
tags:
- practice
- databases
---
Loads neon before the Neon CLI or a Neon MCP tool is used, so the branch-first workflow and primitives guidance apply.
