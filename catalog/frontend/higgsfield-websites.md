---
name: higgsfield-websites
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?)?higgsfield\s+(-\S+\s+)*website\b
action:
  type: require-skill
  skill: higgsfield-websites
upstream:
  repo: github.com/higgsfield-ai/skills
  path: higgsfield-websites/SKILL.md
  ref: 27000402070db19d958789d05b3b52b6f4718d7c
  content_hash: sha256:03f247a262133675f6a573fb6c66f1be8950f4addde18c5fa3499c39dd32ea64
provenance:
  stars: 1949
  license: MIT
  last_push: '2026-11-05'
  fetched: '2026-11-12'
summary: Builds, edits and deploys React/TanStack sites, apps and games on Higgsfield, including game art.
tags:
- practice
- frontend
---
Loads higgsfield-websites before a Higgsfield website is created or published, so the product type and deploy gates are followed.
