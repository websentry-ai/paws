---
name: prisma-orm-setup
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?prisma(@\S+)?\s+(init|db\s+(pull|push))($|[^\w-])
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
tags:
- practice
- databases
---
Loads prisma-orm-setup before Prisma is initialized, installed or connected to a database, so the right Prisma version and provider setup are chosen.
