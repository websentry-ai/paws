---
name: "stop-slop-mcp-send"
event: "pre-tool"
matcher:
  mcp_action: "send"
action:
  type: require-skill
  skill: "stop-slop"
---
Any MCP send: Slack and Gmail messages, drafts, Notion and Linear comments
