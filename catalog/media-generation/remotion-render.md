---
name: remotion-render
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?remotion(@\S+)?\s+(render|still|benchmark|(lambda|cloudrun)\s+render)($|[^\w-])
action:
  type: require-skill
  skill: remotion-render
upstream:
  repo: github.com/remotion-dev/skills
  path: skills/remotion-render/SKILL.md
  ref: 1af7e69a2a613437d46095924057970a6a7df40d
  content_hash: sha256:781b458d874c7c8037a84f0b5e8de0cb38ec477db4e0d821b2d6f5de144c1bfa
provenance:
  stars: 6813
  license: NOASSERTION
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Renders Remotion videos and stills with the bundled browser and the right render flags.
tags:
- practice
- media-generation
---
Loads remotion-render before a Remotion render, so the bundled browser and correct render flags are used.
