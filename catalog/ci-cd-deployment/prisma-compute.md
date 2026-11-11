---
name: prisma-compute
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?(prisma|@prisma/cli)(@\S+)?\s+(deploy|service|git\s+connect|project\s+(env|link)|app\s+deploy)($|[^\w-])
action:
  type: require-skill
  skill: prisma-compute
upstream:
  repo: github.com/prisma/skills
  path: prisma-compute/SKILL.md
  ref: a6f192959cf43b08aa330fe1ffca090267215812
  content_hash: sha256:c5bc4413f1e30e464e20b23a12548c7136beb1aa0fbf51fd688342de3f838e88
provenance:
  stars: 80
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Deploys and operates apps on Prisma Compute with the Prisma 8 CLI: deploy, preview stages, logs, rollback, env, domains.'
tags:
- practice
- ci-cd-deployment
---
Loads prisma-compute before a Prisma Compute deploy or service command, so the Prisma 8 CLI surface, stages and consent tokens are used.
