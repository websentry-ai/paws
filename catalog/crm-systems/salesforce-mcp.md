---
name: salesforce-mcp
event: pre-tool
matcher:
  tool: mcp__salesforce__*
  mcp_action: create|update|delete
action:
  type: require-skill
  skill: salesforce
  on_miss: digest
  digest: digests/salesforce.md
tags: [org, crm]
---
Loads the salesforce skill before a Salesforce MCP record create, update or delete.
