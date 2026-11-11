---
name: lark-attendance
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+attendance\b
action:
  type: require-skill
  skill: lark-attendance
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-attendance/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:3a4887f214eae0d3ea353c68039981ff86011220c5f5bfe21158f8e924c1c02f
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Looks up the user's own Lark/Feishu attendance clock-in records via lark-cli.
tags:
- practice
- workplace-suite
---
Loads lark-attendance before lark-cli queries attendance, so the required default parameters are filled in.
