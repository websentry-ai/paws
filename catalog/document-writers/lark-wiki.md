---
name: lark-wiki
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+wiki\b
action:
  type: require-skill
  skill: lark-wiki
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-wiki/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:d0497ef3a8ff645e3fe22fa1cdd92f090760a5cd605e91d05e06b5da70a5f06e
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Manages Lark/Feishu wiki spaces, members and node hierarchy through lark-cli.
tags:
- practice
- document-writers
---
Loads lark-wiki before lark-cli manages wiki spaces, members or nodes, so node moves and permissions are handled correctly.
