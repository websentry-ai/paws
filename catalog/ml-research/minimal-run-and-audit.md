---
name: minimal-run-and-audit
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/repro_outputs/**/*.{md,json}'
action:
  type: require-skill
  skill: minimal-run-and-audit
upstream:
  repo: github.com/lllllllama/rigorpilot-skills
  path: skills/minimal-run-and-audit/SKILL.md
  ref: fb3ccdf5aa64b1fa5b1d0463e666a8bc8bc26e01
  content_hash: sha256:a0f4f65331c86d62ea7e5c22fe6014c96592acb7817e4fe7c5f1de7d50d8f4e8
provenance:
  stars: 486
  license: MIT
  last_push: '2026-09-23'
  fetched: '2026-11-12'
summary: Captures evidence from a smoke, inference or eval run and writes standardized repro_outputs/ files with patch notes.
tags:
- practice
- ml-research
---
Loads minimal-run-and-audit before repro_outputs/ reports are written, so run evidence follows the normalization and patch-note rules.
