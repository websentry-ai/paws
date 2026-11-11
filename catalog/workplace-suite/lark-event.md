---
name: lark-event
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+event\b
action:
  type: require-skill
  skill: lark-event
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-event/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:87fdef00c75d9a21e47c65298299c9f868ff381edaf8c0f53f3d2724a8fb134e
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Streams Lark/Feishu real-time events (messages, approvals, tasks, meetings) as NDJSON with lark-cli event consume.
tags:
- practice
- workplace-suite
---
Loads lark-event before lark-cli subscribes to events, so ready markers and bounded runs are used.
