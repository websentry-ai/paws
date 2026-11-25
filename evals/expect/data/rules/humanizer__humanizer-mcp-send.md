---
name: humanizer-mcp-send
event: pre-tool
matcher:
  mcp_action: send
action:
  type: require-skill
  skill: humanizer
upstream:
  repo: github.com/blader/humanizer
  path: SKILL.md
  ref: 8c307f367c91bd974002f10f2be2965d642c13df
  content_hash: sha256:f88f72df3adf966a49f136648dae5fc9eb009f778e62bca777c26603e7f63fb7
provenance:
  stars: 61292
  license: MIT
  last_push: 2026-10-16
  fetched: 2026-10-27
tags: [practice, writing]
---
Loads humanizer before any MCP send: Slack and Gmail messages, drafts, Notion
and Linear comments, GitHub issue comments.
