---
name: run-train
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/train_outputs/**/*.{md,json}'
action:
  type: require-skill
  skill: run-train
upstream:
  repo: github.com/lllllllama/rigorpilot-skills
  path: skills/run-train/SKILL.md
  ref: fb3ccdf5aa64b1fa5b1d0463e666a8bc8bc26e01
  content_hash: sha256:5a6cfa45e2eab2aae43333806255742bf21d580f030953552353bdc47d11a8d2
provenance:
  stars: 486
  license: MIT
  last_push: '2026-09-23'
  fetched: '2026-11-12'
summary: Runs a selected training command conservatively (startup, short run, kickoff, resume) and records it in train_outputs/.
tags:
- practice
- ml-research
---
Loads run-train before train_outputs/ evidence files are written, so command, seed, log and metric records follow the standard layout.
