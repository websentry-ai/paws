---
name: higgsfield-marketplace-cards
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?)?higgsfield\s+(-\S+\s+)*marketplace-cards\b
action:
  type: require-skill
  skill: higgsfield-marketplace-cards
upstream:
  repo: github.com/higgsfield-ai/skills
  path: higgsfield-marketplace-cards/SKILL.md
  ref: 27000402070db19d958789d05b3b52b6f4718d7c
  content_hash: sha256:dc920d81d0842e0f5dfb0c22ab3b96b16fb75ae62d33b1b8a3a9529d48d2af6f
provenance:
  stars: 1949
  license: MIT
  last_push: '2026-11-05'
  fetched: '2026-11-12'
summary: Generates marketplace listing images (main, secondary, A+ modules) through the Higgsfield CLI.
tags:
- practice
- media-generation
---
Loads higgsfield-marketplace-cards before listing cards are created, so scope and asset choices follow the marketplace rules.
