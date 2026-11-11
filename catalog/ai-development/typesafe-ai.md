---
name: typesafe-ai
event: pre-tool
matcher:
  tool: WebFetch
  command_regex: ^https?://(docs|api)\.typesafe\.ai($|[/?#])
action:
  type: require-skill
  skill: typesafe-ai
upstream:
  repo: github.com/typesafe-ai/skills
  path: skills/typesafe-ai/SKILL.md
  ref: 302eea662c54bca6730873fe245bec1d69ebc23d
  content_hash: sha256:a6d55f954bdee91ffcb208c42fe31cfba71594bc4d6bf63c0fed22bdb81df3d9
provenance:
  stars: 2892
  license: MIT
  last_push: '2026-10-23'
  fetched: '2026-11-12'
summary: 'Integrates TypeSafe''s Jev model: writes typed questions and state, replacing prompt-and-parse steps.'
tags:
- practice
- ai-development
---
Loads typesafe-ai before TypeSafe docs or API schema are fetched, so typed questions and state are written the supported way.
