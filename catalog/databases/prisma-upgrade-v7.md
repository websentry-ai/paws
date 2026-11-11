---
name: prisma-upgrade-v7
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npm|pnpm|yarn|bun)\s+(i|install|add|up|update|upgrade)\b[^\n;&|]*\s(prisma|@prisma/client)@[\^~]?7($|[^\w-])
action:
  type: require-skill
  skill: prisma-upgrade-v7
upstream:
  repo: github.com/prisma/skills
  path: prisma-upgrade-v7/SKILL.md
  ref: a6f192959cf43b08aa330fe1ffca090267215812
  content_hash: sha256:f6e8519788f89e6ee404091eba28236af67551e450f100c9595b208d840770d9
provenance:
  stars: 80
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Migrates a Prisma 6 project to 7: prisma-client generator, driver adapters, prisma.config.ts, ESM and env loading.'
tags:
- practice
- databases
---
Loads prisma-upgrade-v7 before Prisma version 7 is installed, so the prisma-client generator, driver adapter and config changes are applied.
