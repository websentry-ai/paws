---
name: lark-shared
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+(auth|config)\b
action:
  type: require-skill
  skill: lark-shared
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-shared/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:9c2a2dd16dec84f0d4e70b8feffac19398b03f70dcd962fc402c3fab56070d57
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Handles lark-cli auth: login, user vs bot identity, scopes, permission errors and _notice output.'
tags:
- practice
- identity-access
---
Loads lark-shared before lark-cli auth or config commands run, so identity, scopes and the high-risk gate are handled correctly.
