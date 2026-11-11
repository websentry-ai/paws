---
name: writing-skills
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/SKILL.md'
action:
  type: require-skill
  skill: writing-skills
upstream:
  repo: github.com/obra/superpowers
  path: skills/writing-skills/SKILL.md
  ref: 22f0b463e7a5e579904dd6211a520321035526f7
  content_hash: sha256:50d87a4c2c6a9d60156ccfdbe844e0a11678968157e494a8a8c02377c273483a
provenance:
  stars: 315870
  license: MIT
  last_push: '2026-11-03'
  fetched: '2026-11-12'
summary: 'Creates and edits agent skills test-first: write a failing scenario, then the SKILL.md that fixes it.'
tags:
- practice
- agent-environment
---
Loads writing-skills before a SKILL.md is written or edited, so the skill is built from a failing baseline scenario.
