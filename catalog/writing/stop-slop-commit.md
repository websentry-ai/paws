---
name: stop-slop-commit
event: pre-commit
matcher:
  tool: Bash
action:
  type: require-skill
  skill: stop-slop
upstream:
  repo: github.com/hardikpandya/stop-slop
  path: SKILL.md
  ref: 8da1f030185bdfe8471220585162991eaeb970e9
  content_hash: sha256:02a5b429945020ed8a5ed4024d57384258c2596f09ab42c3e7498a8a43ec1c36
provenance:
  stars: 18395
  license: MIT
  last_push: 2026-03-17
  fetched: 2026-10-27
tags: [practice, writing]
---
Loads stop-slop before a commit message is written.
