---
name: archify
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(node\s+(-\S+\s+)*(\S*/)?archify\.mjs|(npx\s+(-y\s+)?)?archify(@\S+)?)\s+[a-z]
action:
  type: require-skill
  skill: archify
upstream:
  repo: github.com/tt-a1i/archify
  path: archify/SKILL.md
  ref: 7e45f36c66de1254d5ad6ba1d878f6499ea269fd
  content_hash: sha256:dcc4821a0dd36701d56e571910de4e20ca99a036810a3ba4d4261b0f078ad7a1
provenance:
  stars: 95446
  license: Apache-2.0
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: Creates validated architecture, sequence, ERD and other diagrams as standalone themed HTML with export.
tags:
- practice
- diagramming
---
Loads archify before the archify CLI runs, so diagrams follow its type schemas and finalize gates.
