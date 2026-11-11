---
name: prisma-orm-setup-2
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npm|pnpm|yarn|bun)\s+(i|install|add)\b[^;&|\n]*(\sprisma(@\S+)?|\s@prisma/(client|adapter-[\w-]+)(@\S+)?)($|[\s;&|)])
action:
  type: require-skill
  skill: prisma-orm-setup
upstream:
  repo: github.com/prisma/skills
  path: prisma-orm-setup/SKILL.md
  ref: a6f192959cf43b08aa330fe1ffca090267215812
  content_hash: sha256:04fe23d99323d5f97675b5038d7d42cec0b5d0ff5871929bd73412afb7941218
provenance:
  stars: 80
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Sets up Prisma ORM or repairs its database connection, choosing the right Prisma version and provider.
tags:
- practice
- databases
---
Loads prisma-orm-setup before Prisma is initialized, installed or connected to a database, so the right Prisma version and provider setup are chosen.
