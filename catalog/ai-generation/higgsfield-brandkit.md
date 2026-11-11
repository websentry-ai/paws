---
name: higgsfield-brandkit
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(python3?|uv\s+run(\s+python3?)?)\s+(-\S+\s+)*\S*scripts/brandkit\.py[\"']?(\s|$)
action:
  type: require-skill
  skill: higgsfield-brandkit
upstream:
  repo: github.com/higgsfield-ai/skills
  path: higgsfield-brandkit/SKILL.md
  ref: 27000402070db19d958789d05b3b52b6f4718d7c
  content_hash: sha256:e7647cd6f6e20426eeafc11b737c0ff4bbcc33776efc29131b2e728500845c33
provenance:
  stars: 1949
  license: MIT
  last_push: '2026-11-05'
  fetched: '2026-11-12'
summary: Builds brand systems (palette, logo, type, mockups, brandbook) with the Higgsfield CLI and a bundled brandkit script.
tags:
- practice
- ai-generation
---
Loads higgsfield-brandkit before the bundled brandkit script runs, so approvals and state follow the skill's workflow.
