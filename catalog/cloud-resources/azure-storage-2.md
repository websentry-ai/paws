---
name: azure-storage-2
event: pre-tool
matcher:
  tool: mcp__*azure*__storage*
action:
  type: require-skill
  skill: azure-storage
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-storage/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:f3ce4441a11ad18f35acb93a0401636fdceb033dbad34c43a3ac9384304bfe77
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Works with Azure Blob, Files, Queue, Table and Data Lake storage, tiers and access via MCP or az storage.
tags:
- practice
- cloud-resources
---
Loads azure-storage before az storage, azcopy or the storage MCP tool is used, so tier and access guidance is applied to the work.
