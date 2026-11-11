---
name: lark-im
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+im\b
action:
  type: require-skill
  skill: lark-im
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-im/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:d79366a9b2a16613f1d930e5563c016d483b05e28e95cbcd7772fdce98503d6f
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Sends and replies to Lark/Feishu messages, searches chats, manages groups, files, reactions and interactive cards.
tags:
- practice
- communications
---
Loads lark-im before lark-cli messages or manages chats, so sends and card handling follow the skill.
