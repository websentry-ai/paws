---
name: humanizer-pr
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*gh\s+pr\s+(create|edit)\b
action:
  type: require-skill
  skill: humanizer
upstream:
  repo: github.com/blader/humanizer
  path: SKILL.md
  ref: 8c307f367c91bd974002f10f2be2965d642c13df
  content_hash: sha256:f88f72df3adf966a49f136648dae5fc9eb009f778e62bca777c26603e7f63fb7
provenance:
  stars: 61292
  license: MIT
  last_push: 2026-10-16
  fetched: 2026-10-27
tags: [practice, writing]
---
Loads humanizer before a PR title or body goes out, so the PR reads like a
person wrote it.
