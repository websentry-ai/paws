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
summary: 'Claude drafts emails instead of sending them, checks recipient domains first, replies in the existing thread, says how many people a message reaches, and writes as itself.'
tags: [org, comms]
---
Loads the gmail skill before the agent sends, drafts, replies to, forwards or labels mail, so external recipients, threads and voice are checked.
