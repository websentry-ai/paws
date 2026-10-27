---
name: gmail-mcp
event: pre-tool
matcher:
  tool: mcp__gmail__create_draft|mcp__gmail__label_message|mcp__gmail__label_thread|mcp__gmail__apply_sensitive_message_label
action:
  type: require-skill
  skill: gmail
  on_miss: digest
  digest: digests/gmail.md
tags: [org, comms]
---
Loads the gmail skill before the agent drafts or labels mail, so external recipients, threads and voice are checked.
