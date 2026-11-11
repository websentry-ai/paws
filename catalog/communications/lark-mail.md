---
name: lark-mail
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+mail\b
action:
  type: require-skill
  skill: lark-mail
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-mail/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:6adfa6cf787c61cfc57fd8b013b6d2c930cb3aa292a7c1cddea88f6cb4091d77
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Drafts, sends, replies to, forwards and searches Lark/Feishu mail, folders, labels and rules via lark-cli.
tags:
- practice
- communications
---
Loads lark-mail before lark-cli drafts, sends or searches mail, so sends and drafts follow the skill.
