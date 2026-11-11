---
name: prisma-cli
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?prisma(@\S+)?\s+(generate|migrate|dev|studio|mcp|validate|format|db\s+seed|db\s+execute|db\s+push\b[^;&|\n]*--(force-reset|accept-data-loss))($|[^\w-])
action:
  type: require-skill
  skill: prisma-cli
upstream:
  repo: github.com/prisma/skills
  path: prisma-cli/SKILL.md
  ref: a6f192959cf43b08aa330fe1ffca090267215812
  content_hash: sha256:5867232acad55272d7278be9144b2cb8dcadd500d5a60a89260a2332bf70bd62
provenance:
  stars: 80
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Prisma ORM 7 CLI reference: init, generate, migrate, db push/pull, dev, studio and mcp.'
tags:
- practice
- databases
---
Loads prisma-cli before a Prisma generate, migrate or reset command, so the Prisma 7 command forms and AI consent rules are followed.
