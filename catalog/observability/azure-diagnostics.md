---
name: azure-diagnostics
event: pre-tool
matcher:
  tool: mcp__*azure*__applens*|mcp__*azure*__resourcehealth*
action:
  type: require-skill
  skill: azure-diagnostics
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-diagnostics/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:c6346495b6abd888e219c8616ee11ac025595fdfa11cfbf37060749c60763222
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
tags:
- practice
- observability
---
Loads azure-diagnostics before AppLens, resource health or Azure log reads, so triage follows the diagnosis flow and checks health first.
