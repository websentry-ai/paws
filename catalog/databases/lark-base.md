---
name: lark-base
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+base\b
action:
  type: require-skill
  skill: lark-base
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-base/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:a92bfeca0654b3eb908bbe1639e13794d4c5c873249408c6bcf9f2c7a163d8d8
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Builds and edits Lark/Feishu Base (multi-dimensional tables): tables, fields, records, views, dashboards and workflows.'
tags:
- practice
- databases
---
Loads lark-base before lark-cli works on a Base, so entity resolution and field or record rules are followed.
