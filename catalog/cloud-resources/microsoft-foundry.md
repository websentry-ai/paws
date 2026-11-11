---
name: microsoft-foundry
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?azd\s+ai(\s|$)
action:
  type: require-skill
  skill: microsoft-foundry
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/microsoft-foundry/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:2e4502618563c5418b3bc83f47adef43fb9cb725348606a5d8331be3dfde1ab3
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
tags:
- practice
- cloud-resources
---
Loads microsoft-foundry before an azd ai agent command or Foundry MCP call, so the dependency check and workflow docs are read first.
