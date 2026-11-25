---
name: "salesforce-mcp"
event: "pre-tool"
matcher:
  tool: "mcp__salesforce__*"
  mcp_action: "create|update|delete"
action:
  type: require-skill
  skill: "salesforce"
---
A Salesforce MCP record create, update or delete
