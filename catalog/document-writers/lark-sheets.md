---
name: lark-sheets
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+sheets\b
action:
  type: require-skill
  skill: lark-sheets
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-sheets/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:14c8c032dc0640e659fbfb9fe56017733b2d1aba6d0664e5ca3cee2628814716
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Creates and edits Lark/Feishu spreadsheets: cells, formulas, styles, charts, pivots and filters via lark-cli.'
tags:
- practice
- document-writers
---
Loads lark-sheets before lark-cli reads or edits a spreadsheet, so ranges, formulas and objects follow the skill.
