---
name: azure-quotas-2
event: pre-tool
matcher:
  tool: mcp__*azure*__quota*
action:
  type: require-skill
  skill: azure-quotas
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-quotas/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:abbbdcd83760090a1d6674d6e0bcac1a83414882deba03861d9539eacb00484c
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Checks Azure quotas and usage per region and requests increases for deployment planning.
tags:
- practice
- cloud-resources
---
Loads azure-quotas before az quota or the quota MCP tool is used, so quota checks follow the CLI-first workflow.
