---
name: extract-design-system
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+(--\s+)?)?extract-design-system\s+(--extract-only|[^-\s])
action:
  type: require-skill
  skill: extract-design-system
upstream:
  repo: github.com/arvindrk/extract-design-system
  path: skills/extract-design-system/SKILL.md
  ref: 1873741ba8dea755e35e6e15134f7918cd58e036
  content_hash: sha256:fcc959a17cec8bd719b97bad240a6a5eaf8660700d4876a4392e4199497d3012
provenance:
  stars: 284
  license: MIT
  last_push: '2026-09-05'
  fetched: '2026-11-12'
summary: Extracts colors, type and spacing from a public website and writes starter design token files.
tags:
- practice
- frontend
---
Loads extract-design-system before extract-design-system is run on a site, so it confirms the target and keeps generated files separate from app styles.
