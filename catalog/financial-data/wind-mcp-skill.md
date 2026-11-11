---
name: wind-mcp-skill
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*node\s+(-\S+\s+)*(\S*/)?cli\.mjs\s+call\s+(stock_data|fund_data|index_data|bond_data|financial_docs|economic_data|analytics_data)($|[^\w-])
action:
  type: require-skill
  skill: wind-mcp-skill
upstream:
  repo: github.com/wind-alice/alicemarket
  path: skills/wind-mcp-skill/SKILL.md
  ref: ab8fa4cfc7904fa28d2a7297cd293d192eda3b5e
  content_hash: sha256:6462fb9a38bc02912fec645474f3304a68313cb30a6bea35aee18479f548a838
provenance:
  stars: 155
  license: NOASSERTION
  last_push: '2026-10-30'
  fetched: '2026-11-12'
summary: Fetches Wind financial market data (stocks, funds, indices, bonds, filings, macro) through a local MCP CLI.
tags:
- practice
- financial-data
---
Loads wind-mcp-skill before a Wind data call, so the right server type, contract and parameters are used.
