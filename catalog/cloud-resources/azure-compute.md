---
name: azure-compute
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?az\s+(vm|vmss)\s+(create|list-skus|list-sizes)\b
action:
  type: require-skill
  skill: azure-compute
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-compute/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:8dff072f62c75b5f649e6db221a2fa5ffde3315d7d11a1577d9bc5e79488b31d
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
tags:
- practice
- cloud-resources
---
Loads azure-compute before a VM or scale set is created or sized, so size recommendation and creation workflows are followed.
