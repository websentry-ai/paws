---
name: azure-reliability
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?az\s+(appservice\s+plan|functionapp(\s+plan)?|webapp|storage\s+account)\s+(create|update)\b[^\n;&|]*(--zone-redundant|Standard_ZRS)\b
action:
  type: require-skill
  skill: azure-reliability
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-reliability/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:2d667d15ea1f2ecd02987d1488bf6476c6468f9fcf5c943af1f25d6985f3a107
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Assesses Functions and App Service reliability (zone redundancy, ZRS, health probes, failover) and stages fixes.
tags:
- practice
- cloud-resources
---
Loads azure-reliability before zone redundancy or ZRS storage is switched on, so the staged remediation and confirmation steps are followed.
