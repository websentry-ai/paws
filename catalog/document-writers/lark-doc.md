---
name: lark-doc
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+(docs|mindnotes)\b
action:
  type: require-skill
  skill: lark-doc
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-doc/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:1323881a7ec6a09345e82929efdbca516f15b41421eef30d5e1e0f29f0e0f8a8
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Reads, creates and edits Lark/Feishu cloud docs and mind notes through lark-cli, including images and attachments.
tags:
- practice
- document-writers
---
Loads lark-doc before lark-cli reads or edits a cloud doc or mind note, so the right workflow reference is used.
