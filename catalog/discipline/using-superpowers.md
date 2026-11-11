---
name: using-superpowers
event: session-start
matcher:
  command_regex: ^(startup|clear|compact)$
action:
  type: require-skill
  skill: using-superpowers
upstream:
  repo: github.com/obra/superpowers
  path: skills/using-superpowers/SKILL.md
  ref: 22f0b463e7a5e579904dd6211a520321035526f7
  content_hash: sha256:6a9104fac7e16c3cec2bc118943aefd331cac35b4cfabd427cea4dc16337c6a5
provenance:
  stars: 315870
  license: MIT
  last_push: '2026-11-03'
  fetched: '2026-11-12'
summary: Requires checking for and invoking relevant skills before any response, including clarifying questions.
tags:
- practice
- discipline
---
Loads using-superpowers when a fresh context starts (startup, clear, compact), so skill checks come before the first reply.
