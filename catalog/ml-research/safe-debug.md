---
name: safe-debug
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/debug_outputs/**/*.{md,json}'
action:
  type: require-skill
  skill: safe-debug
upstream:
  repo: github.com/lllllllama/rigorpilot-skills
  path: skills/safe-debug/SKILL.md
  ref: fb3ccdf5aa64b1fa5b1d0463e666a8bc8bc26e01
  content_hash: sha256:35e8304178e6188d4bc66c0b10f32bb46620fbb7180a446b842865f755e7baf1
provenance:
  stars: 486
  license: MIT
  last_push: '2026-09-23'
  fetched: '2026-11-12'
summary: Diagnoses deep learning failures (tracebacks, OOM, NaN loss, shape mismatch) before patching, with approval for fixes.
tags:
- practice
- ml-research
---
Loads safe-debug before debug_outputs/ diagnosis files are written, so fixes stay separate from research changes and wait for approval.
