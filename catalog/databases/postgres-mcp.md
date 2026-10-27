---
name: postgres-mcp
event: pre-tool
matcher:
  tool: mcp__postgres__execute_sql
  command_regex: (?i:\b(update|delete|insert|drop|truncate|alter|create)\b)
action:
  type: require-skill
  skill: postgres
  on_miss: digest
  digest: digests/postgres.md
tags: [org, database]
---
Loads the postgres skill before the postgres MCP server runs a write or DDL statement.
