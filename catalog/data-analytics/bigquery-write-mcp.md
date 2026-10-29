---
name: bigquery-write-mcp
event: pre-tool
matcher:
  tool: mcp__bigquery__*|mcp__claude_ai_BigQuery__*
  mcp_action: create|update|delete
action:
  type: require-skill
  skill: bigquery
  on_miss: digest
  digest: digests/bigquery.md
tags: [org, data]
---
Loads the bigquery skill before a BigQuery MCP dataset or table create, update or delete.
