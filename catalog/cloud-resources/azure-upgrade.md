---
name: azure-upgrade
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?az\s+(functionapp\s+flex-migration\b|(functionapp\s+plan|appservice\s+plan)\s+update\b[^;&|\n]*--sku\b)
action:
  type: require-skill
  skill: azure-upgrade
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-upgrade/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:730671ae4ad6a09b4d90f69c2825bec4c0b1c04b2e27b3bfb505280918633f63
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Upgrades Azure workloads between plans, tiers or SKUs and modernises legacy Azure SDK dependencies.
tags:
- practice
- cloud-resources
---
Loads azure-upgrade before a Functions flex migration or a plan SKU change, so the assessment and upgrade steps are followed.
