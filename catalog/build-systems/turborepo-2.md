---
name: turborepo-2
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?turbo(@\S+)?\s+(prune|boundaries|query|gen|link|unlink|login|logout)($|[^\w-])
action:
  type: require-skill
  skill: turborepo
upstream:
  repo: github.com/vercel/turborepo
  path: skills/turborepo/SKILL.md
  ref: af729aba869736a9f03a0192d438cca35f4cc02b
  content_hash: sha256:49bb6427015253ce5d93f15dfa6c9ade3f8528f0fbe54769b6da1e1458f058d4
provenance:
  stars: 31326
  license: MIT
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: Configures and troubleshoots Turborepo from the docs bundled in the installed turbo package.
tags:
- practice
- build-systems
---
Loads turborepo before turbo.json is edited or a Turborepo config command runs, so the docs bundled with the installed turbo version are read first.
