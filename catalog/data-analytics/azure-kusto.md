---
name: azure-kusto
event: pre-tool
matcher:
  tool: mcp__*__kusto*
action:
  type: require-skill
  skill: azure-kusto
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-kusto/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:bc5ead622d2b3957f486ec9a5999e042b6a4192422ec690cdead11bb65797062
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
tags:
- practice
- data-analytics
---
Loads azure-kusto before a Kusto MCP call or az kusto command, so KQL patterns and schema exploration steps are followed.
