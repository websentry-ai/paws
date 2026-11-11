---
name: prisma-postgres-setup
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?create-db($|[^\w-])
action:
  type: require-skill
  skill: prisma-postgres-setup
upstream:
  repo: github.com/prisma/skills
  path: prisma-postgres-setup/SKILL.md
  ref: a6f192959cf43b08aa330fe1ffca090267215812
  content_hash: sha256:0cb1de49d7d9a73dde137522ba8359a56f154630ff228e9c058655457abeece0
provenance:
  stars: 80
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Gets or reuses a Prisma Postgres database, connects the app and verifies it, handing ORM config to prisma-orm-setup.
tags:
- practice
- databases
---
Loads prisma-postgres-setup before create-db provisions a Prisma Postgres database, so an existing connection is reused first.
