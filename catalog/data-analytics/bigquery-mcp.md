---
name: bigquery-mcp
event: pre-tool
matcher:
  tool: mcp__bigquery__*|mcp__claude_ai_BigQuery__*
  command_regex: (?i)\b(select|with|insert|update|delete|merge|create|drop|alter|truncate)\b
action:
  type: require-skill
  skill: bigquery
  on_miss: digest
  digest: digests/bigquery.md
tags: [org, data]
---
Loads the bigquery skill before a BigQuery MCP tool runs SQL (cost preview first), whatever the tool is called.
