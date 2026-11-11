---
name: azure-resource-lookup
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?az\s+(graph\s+query|resource\s+list)\b
action:
  type: require-skill
  skill: azure-resource-lookup
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-resource-lookup/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:2ecc776e2fc090bbe0b5c198cc560803e477feb6fc0387def7747b061e3b358b
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Lists and finds Azure resources across subscriptions and groups, using Resource Graph when MCP tools fall short.
tags:
- practice
- cloud-resources
---
Loads azure-resource-lookup before az graph query or az resource list runs, so Resource Graph routing and query patterns are used.
