---
name: analyze-project
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/analysis_outputs/**/*.{md,json}'
action:
  type: require-skill
  skill: analyze-project
upstream:
  repo: github.com/lllllllama/rigorpilot-skills
  path: skills/analyze-project/SKILL.md
  ref: fb3ccdf5aa64b1fa5b1d0463e666a8bc8bc26e01
  content_hash: sha256:ceff36812339e36f56d0c27f92c7e01782f1afdb580b74da4498779883c9bd92
provenance:
  stars: 486
  license: MIT
  last_push: '2026-09-23'
  fetched: '2026-11-12'
summary: 'Read-only analysis of a deep learning repo: model structure, entrypoints, configs, suspicious patterns.'
tags:
- practice
- ml-research
---
Loads analyze-project before analysis_outputs/ reports are written, so findings stay read-only and flagged as heuristics.
