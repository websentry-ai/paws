---
name: find-skills
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?skills(@\S+)?\s+(find|add)($|[^\w-])
action:
  type: require-skill
  skill: find-skills
upstream:
  repo: github.com/vercel-labs/skills
  path: skills/find-skills/SKILL.md
  ref: e781cb4d07433dfe1d19e924ed45cc61738c55ec
  content_hash: sha256:7bc223e2aafbdbf6e50ea243ba6ef67fe432bb069ab439fea0390795dac0bf7f
provenance:
  stars: 38910
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Searches the skills.sh ecosystem with the skills CLI, checks install counts and quality, then installs the right skill.
tags:
- practice
- agent-environment
---
Loads find-skills before the skills CLI searches for or installs a skill, so quality and install counts are checked first.
