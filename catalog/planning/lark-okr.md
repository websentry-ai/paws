---
name: lark-okr
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+okr\b
action:
  type: require-skill
  skill: lark-okr
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-okr/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:54c97e28bea818a3adb09d19723c81af03450475a0b7fe46fdc524b856dee761
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Views and edits Lark/Feishu OKR cycles, objectives, key results, alignments and progress via lark-cli.
tags:
- practice
- planning
---
Loads lark-okr before lark-cli views or edits OKRs, so cycles, objectives and progress follow the skill.
