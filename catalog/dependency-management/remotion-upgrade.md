---
name: remotion-upgrade
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?remotion(@\S*)?\s+upgrade($|[^\w-])
action:
  type: require-skill
  skill: remotion-upgrade
upstream:
  repo: github.com/remotion-dev/skills
  path: skills/remotion-upgrade/SKILL.md
  ref: 1af7e69a2a613437d46095924057970a6a7df40d
  content_hash: sha256:9aec7684e1b4bd42e343f9b6d9757bb167c0ae8293f41f42280caade73ea975b
provenance:
  stars: 6813
  license: NOASSERTION
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Upgrades remotion and all @remotion/* packages to one version, aligns helper deps, and updates Remotion skills.
tags:
- practice
- dependency-management
---
Loads remotion-upgrade before Remotion packages are upgraded, so every @remotion package lands on one version.
