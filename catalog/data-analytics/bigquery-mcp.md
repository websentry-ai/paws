---
name: bigquery-mcp
event: pre-tool
matcher:
  tool: mcp__bigquery__execute_query|mcp__bigquery__create_dataset|mcp__bigquery__delete_table
action:
  type: require-skill
  skill: bigquery
  on_miss: digest
  digest: digests/bigquery.md
tags: [org, data]
---
Loads the bigquery skill before a BigQuery MCP query, dataset create or table delete.
