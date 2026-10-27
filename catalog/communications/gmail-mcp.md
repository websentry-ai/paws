---
name: gmail-mcp
event: pre-tool
matcher:
  tool: mcp__gmail__*|mcp__claude_ai_Gmail__*
  mcp_action: send|update
action:
  type: require-skill
  skill: gmail
  on_miss: digest
  digest: digests/gmail.md
tags: [org, comms]
---
Loads the gmail skill before the agent sends, drafts, replies to, forwards or labels mail, so external recipients, threads and voice are checked.
