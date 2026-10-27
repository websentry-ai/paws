---
name: slack-mcp
event: pre-tool
matcher:
  tool: mcp__slack__post_message|mcp__slack__schedule_message|mcp__slack__upload_file
action:
  type: require-skill
  skill: slack
  on_miss: digest
  digest: digests/slack.md
tags: [org, comms]
---
Loads the slack skill before a Slack MCP send, scheduled send or file upload.
