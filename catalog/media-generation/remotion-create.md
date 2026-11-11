---
name: remotion-create
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*((npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?create-video|(npm|pnpm|yarn|bun)\s+(init|create)\s+video)($|[^\w-])
action:
  type: require-skill
  skill: remotion-create
upstream:
  repo: github.com/remotion-dev/skills
  path: skills/remotion-create/SKILL.md
  ref: 1af7e69a2a613437d46095924057970a6a7df40d
  content_hash: sha256:3e453c3d1f019d9fd4269d0dc523b56546ec77fc004e7dc65f13293293927300
provenance:
  stars: 6813
  license: NOASSERTION
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Scaffolds a new Remotion project and first composition in the right folder.
tags:
- practice
- media-generation
---
Loads remotion-create before a Remotion project is scaffolded, so it lands in the right folder with a first composition.
