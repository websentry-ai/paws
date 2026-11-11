---
name: uizze-ui
event: pre-tool
matcher:
  tool: mcp__*uizze*__*
action:
  type: require-skill
  skill: uizze-ui
upstream:
  repo: github.com/uizze/uizze
  path: skills/uizze-ui/SKILL.md
  ref: fc401e9e654cbafe427b0c98142fc8bccd0c4e7e
  content_hash: sha256:f06dec4c69d265bd48487594cb5e663d5f5c0ea7bc96d568d82adcf6f6ed2bef
provenance:
  stars: 95
  license: MIT
  last_push: '2026-11-01'
  fetched: '2026-11-12'
summary: Builds and improves UI from the project's code and assets, drawing on uizze.com examples, design systems and fonts.
tags:
- practice
- frontend
---
Loads uizze-ui before a Uizze MCP search or retrieval call, so queries, selection and font and icon licensing follow its rules.
