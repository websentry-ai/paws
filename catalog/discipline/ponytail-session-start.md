---
name: ponytail-session-start
event: session-start
matcher:
  command_regex: ^(startup|clear|compact)$
action:
  type: require-skill
  skill: ponytail
upstream:
  repo: github.com/DietrichGebert/ponytail
  path: skills/ponytail/SKILL.md
  ref: 09c434ab73ca43761c5d8b0c5d341926c2fba710
  content_hash: sha256:56d72677aca02689ac89bd48734e8c19b40ab86716d1257eaf5f1238b43cda6b
provenance:
  stars: 178631
  license: MIT
  last_push: 2026-10-18
  fetched: 2026-10-28
tags: [practice, discipline]
---
Names ponytail at the start of every fresh context (startup, clear, compact) so the agent takes the smallest complete change from the first edit.
