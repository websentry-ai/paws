---
name: skill-creator
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/SKILL.md'
action:
  type: require-skill
  skill: skill-creator
upstream:
  repo: github.com/anthropics/skills
  path: skills/skill-creator/SKILL.md
  ref: 2c5958ecac7cee2b583967a2c4981d0f4a75c579
  content_hash: sha256:2d7f889e6ae93c6c27d699856942c336b5b8644537496987d5571043d059b1dc
provenance:
  stars: 192604
  license: NOASSERTION
  last_push: '2026-11-06'
  fetched: '2026-11-12'
summary: Drafts, tests and iterates on skills with eval runs, benchmarks and description tuning for better triggering.
tags:
- practice
- agent-environment
---
Loads skill-creator before a SKILL.md is written or edited, so descriptions and structure follow the skill-authoring process.
