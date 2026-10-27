---
name: frontend-design-ui
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/*.{tsx,jsx,vue,svelte,css,scss}'
action:
  type: require-skill
  skill: frontend-design
upstream:
  repo: github.com/anthropics/skills
  path: skills/frontend-design/SKILL.md
  ref: dbd4588f9e1033efb41dad4bef2f7947c8993d44
  content_hash: sha256:83485963700bcf11d5bf78118facb301676b69d918fee2e45a95e7f1a2840f6c
provenance:
  stars: 184140
  license: Apache-2.0
  last_push: 2026-10-09
  fetched: 2026-10-27
tags: [practice, frontend]
---
Loads Anthropic's frontend-design before a component or stylesheet changes, so the UI gets a deliberate visual direction instead of template defaults.
