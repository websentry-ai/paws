---
name: shadcn
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?shadcn(-ui)?(@\S+)?\s+(init|add|apply|preset|search|info|docs|view|diff|migrate|build|mcp|list)($|[^\w-])
action:
  type: require-skill
  skill: shadcn
upstream:
  repo: github.com/shadcn/ui
  path: skills/shadcn/SKILL.md
  ref: 8ab27715521cbd4f87ae6886cd989cbb678f6de7
  content_hash: sha256:388962d16c7a4f13aaeee0d8f128e50a87df8553d42ef7988c39c0e1c74a40e8
provenance:
  stars: 127877
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
tags:
- practice
- frontend
---
Loads shadcn before the shadcn CLI runs or components.json is edited, so project config, registries and component rules are followed.
