---
name: lark-contact
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+contact\b
action:
  type: require-skill
  skill: lark-contact
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-contact/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:0073e5dd26bc5d1b76a663523ef80f2caa89d8011edbeef6dc61fe8d274ab624
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Resolves Lark/Feishu people by name or email to open_id and looks up profiles and bots via lark-cli.
tags:
- practice
- communications
---
Loads lark-contact before lark-cli looks up people, so names resolve to the right open_id.
