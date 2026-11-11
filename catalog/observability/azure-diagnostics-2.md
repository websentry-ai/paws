---
name: azure-diagnostics-2
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?az\s+(monitor\s+activity-log\s+list|containerapp\s+logs\s+show|webapp\s+log\s+(tail|download|show))\b
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
summary: Diagnoses Azure production issues with AppLens, Azure Monitor, resource health and activity logs.
tags:
- practice
- observability
---
Loads azure-diagnostics before AppLens, resource health or Azure log reads, so triage follows the diagnosis flow and checks health first.
