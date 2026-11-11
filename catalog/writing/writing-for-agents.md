---
name: writing-for-agents
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/{AGENTS,CLAUDE}.md'
action:
  type: require-skill
  skill: writing-for-agents
upstream:
  repo: github.com/mattpocock/skills
  path: skills/productivity/writing-for-agents/SKILL.md
  ref: 4ae2317ddb22b046e26455e4f9c9b3aaaf62212d
  content_hash: sha256:e79a2476b2417b228922453478d9505a6b9af19f3a412e560728ce62ed3555ca
provenance:
  stars: 321219
  license: MIT
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: 'Guides writing AGENTS.md and CLAUDE.md: sharp trigger pointers, low context load, clear steps versus reference.'
tags:
- practice
- writing
---
Loads writing-for-agents before AGENTS.md or CLAUDE.md is written or edited, so agent instructions stay short and trigger-first.
