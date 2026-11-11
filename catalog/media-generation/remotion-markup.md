---
name: remotion-markup
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/remotion/**/*.{tsx,jsx}'
action:
  type: require-skill
  skill: remotion-markup
upstream:
  repo: github.com/remotion-dev/skills
  path: skills/remotion-markup/SKILL.md
  ref: 1af7e69a2a613437d46095924057970a6a7df40d
  content_hash: sha256:e4b1d9451aa08c62e4901d8f1f8323d24133bbb108dcea819a4a1690defb14cb
provenance:
  stars: 6813
  license: NOASSERTION
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: 'Best practices for Remotion React markup: frame-scoped components, timing, animation and effects.'
tags:
- practice
- media-generation
---
Loads remotion-markup before a Remotion component is written, so frame-scoped timing and animation rules are followed.
