---
name: humanize-sends
event: pre-tool
matcher:
  mcp_action: send
action:
  type: require-skill
  skill: outbound-voice
---
Any MCP send (chat, mail, comments) loads the outbound-voice skill first.
