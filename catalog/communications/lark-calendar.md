---
name: lark-calendar
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+calendar\b
action:
  type: require-skill
  skill: lark-calendar
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-calendar/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:afb5a989579992d5427207df5190e35934a1f187ff72b45bb731749c55087553
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Views and edits Lark/Feishu calendar events, attendees, free/busy and meeting room bookings through lark-cli.
tags:
- practice
- communications
---
Loads lark-calendar before lark-cli reads or edits calendar events, so scheduling and room booking follow the skill.
