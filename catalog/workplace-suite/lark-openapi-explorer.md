---
name: lark-openapi-explorer
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+api($|[^\w-])
action:
  type: require-skill
  skill: lark-openapi-explorer
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-openapi-explorer/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:a6de3d251fba0ebb4368da53600befa2edd8ab11fd886488b3b197e456e96a4e
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
tags:
- practice
- workplace-suite
---
Loads lark-openapi-explorer before a raw Lark OpenAPI call or doc fetch, so endpoints come from the official docs and not guesses.
