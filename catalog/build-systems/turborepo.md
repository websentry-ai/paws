---
name: turborepo
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/{turbo.json,turbo.jsonc}'
action:
  type: require-skill
  skill: turborepo
upstream:
  repo: github.com/vercel/turborepo
  path: skills/turborepo/SKILL.md
  ref: af729aba869736a9f03a0192d438cca35f4cc02b
  content_hash: sha256:49bb6427015253ce5d93f15dfa6c9ade3f8528f0fbe54769b6da1e1458f058d4
provenance:
  stars: 31326
  license: MIT
  last_push: '2026-11-11'
  fetched: '2026-11-12'
tags:
- practice
- build-systems
---
Loads turborepo before turbo.json is edited or a Turborepo config command runs, so the docs bundled with the installed turbo version are read first.
