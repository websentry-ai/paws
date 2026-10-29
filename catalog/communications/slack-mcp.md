---
name: slack-mcp
event: pre-tool
matcher:
  tool: mcp__slack__*|mcp__plugin_slack_slack__*|mcp__claude_ai_Slack__*
  mcp_action: send
action:
  type: require-skill
  skill: slack
  on_miss: digest
  digest: digests/slack.md
tags: [org, comms]
---
Loads the slack skill before a Slack MCP send, draft or scheduled send.
