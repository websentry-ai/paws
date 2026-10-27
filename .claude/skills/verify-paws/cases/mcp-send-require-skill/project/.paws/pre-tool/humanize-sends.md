---
name: humanize-sends
event: pre-tool
matcher:
  mcp_action: send
action:
  type: require-skill
  skill: humanizer
---
Any MCP send (chat, mail, comments) loads humanizer first.
