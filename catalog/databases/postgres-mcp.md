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
summary: 'Safe Postgres writes: count the rows first, wrap changes in a transaction, and make prod schema changes as migrations.'
tags: [org, database]
---
Loads the postgres skill before the postgres MCP server runs a write or DDL statement.
