---
name: "salesforce-dml-mcp"
event: "pre-tool"
matcher:
  tool: "mcp__salesforce__*"
  command_regex: "(?i)\"operation\"\\s*:\\s*\"(insert|update|delete|upsert)\""
action:
  type: require-skill
  skill: "salesforce"
---
A Salesforce MCP DML call
