---
name: lark-apps
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+apps\b
action:
  type: require-skill
  skill: lark-apps
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-apps/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:635bd203606de56f343e6a0303fc2e5aa0742d589a2f0686b7d83486c61b8b64
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Creates, develops and hosts Lark Miaoda (Spark) apps, locally or in the cloud, via lark-cli apps.
tags:
- practice
- workplace-suite
---
Loads lark-apps before lark-cli creates or hosts a Miaoda app, so the right development path and auth are used.
