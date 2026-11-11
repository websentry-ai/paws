---
name: lark-slides
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+slides\b
action:
  type: require-skill
  skill: lark-slides
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-slides/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:d6b44e76363b4fc64f742cfdf2c42520a19d2e5d6a89a377302c4d50566f6fd5
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Creates and edits Lark/Feishu slide decks page by page through lark-cli, with strict layout and image rules.
tags:
- practice
- document-writers
---
Loads lark-slides before lark-cli creates or edits a slide deck, so layout and image rules are followed.
