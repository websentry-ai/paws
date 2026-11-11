---
name: lark-drive
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+drive\b
action:
  type: require-skill
  skill: lark-drive
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-drive/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:f7af57d7639b503bc4fb562aa27465e390b777d265c36ce50ebc6309258a9601
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Manages Lark/Feishu Drive files and folders via lark-cli: upload, download, move, delete, permissions and imports.'
tags:
- practice
- file-transfer
---
Loads lark-drive before lark-cli touches Drive files or folders, so tokens, imports and permission changes are handled correctly.
