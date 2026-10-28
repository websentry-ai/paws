---
name: caveman-commit-message
event: pre-commit
matcher:
  tool: Bash
action:
  type: require-skill
  skill: caveman-commit
upstream:
  repo: github.com/JuliusBrussee/caveman
  path: skills/caveman-commit/SKILL.md
  ref: 9b68f7ed12b05e7efaff18e65c5a3fc2849fd0a4
  content_hash: sha256:2c1472fb59eedd3cfba0b6512dc470a00287343ade421db55c21a6fb52afbe86
provenance:
  stars: 115660
  license: Apache-2.0
  last_push: 2026-10-28
  fetched: 2026-10-28
tags: [practice, git]
---
Loads caveman-commit before a commit, so the message is a terse Conventional Commits subject and a body only when the why is not obvious.
