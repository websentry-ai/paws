---
name: salesforce-mcp
event: pre-tool
matcher:
  tool: mcp__salesforce__create_record|mcp__salesforce__update_record|mcp__salesforce__delete_record|mcp__salesforce__upsert_record|mcp__salesforce__execute_soql_dml
action:
  type: require-skill
  skill: salesforce
  on_miss: digest
  digest: digests/salesforce.md
tags: [org, crm]
---
Loads the salesforce skill before a Salesforce MCP record write or DML.
