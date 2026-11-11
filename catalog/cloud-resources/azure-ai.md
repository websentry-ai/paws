---
name: azure-ai
event: pre-tool
matcher:
  tool: mcp__*azure__search*|mcp__*azure-mcp__search*|mcp__*azure_mcp__search*|mcp__*azure__speech*|mcp__*azure-mcp__speech*|mcp__*azure_mcp__speech*
action:
  type: require-skill
  skill: azure-ai
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-ai/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:9cc83d04c2d242f64a209ff4c4013f07613c2874d17cf53b86b66d695c6945f2
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
tags:
- practice
- cloud-resources
---
Loads azure-ai before an Azure Search or Speech MCP call or az search and cognitiveservices command, so service patterns and SDK guides are used.
