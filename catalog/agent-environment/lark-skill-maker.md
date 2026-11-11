---
name: lark-skill-maker
event: pre-tool
matcher:
  tool: Write
  file_glob: '**/skills/lark-*/SKILL.md'
action:
  type: require-skill
  skill: lark-skill-maker
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-skill-maker/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:f09348cc73fb4064f3bfabf1dda3af5be989eb591f5a1108d254be7be0885a20
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Writes new custom SKILL.md files that wrap Lark/Feishu API calls or multi-step lark-cli flows.
tags:
- practice
- agent-environment
---
Loads lark-skill-maker before a new lark-* skill file is written, so the skill is built from the real CLI commands.
