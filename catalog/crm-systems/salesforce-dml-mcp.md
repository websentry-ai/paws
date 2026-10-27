---
name: salesforce-dml-mcp
event: pre-tool
matcher:
  tool: mcp__salesforce__*
  command_regex: (?i)"operation"\s*:\s*"(insert|update|delete|upsert)"
action:
  type: require-skill
  skill: salesforce
  on_miss: digest
  digest: digests/salesforce.md
tags: [org, crm]
---
Loads the salesforce skill before a Salesforce MCP DML call (insert, update, delete, upsert), whose tool name carries no write verb.
