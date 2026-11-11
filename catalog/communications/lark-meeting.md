---
name: lark-meeting
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+(vc|minutes|note)\b
action:
  type: require-skill
  skill: lark-meeting
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-meeting/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:dd170533fba15ae81076f5da705d9047547a2b16c529d8608aa06bc8b2c11cd2
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Queries Lark/Feishu meetings, minutes, transcripts and notes, and sends in-meeting chat via lark-cli.
tags:
- practice
- communications
---
Loads lark-meeting before lark-cli queries meetings, minutes or notes, so source identity and artifact routing are right.
