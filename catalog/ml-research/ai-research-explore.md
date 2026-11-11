---
name: ai-research-explore
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/explore_outputs/**/*.{md,json}'
action:
  type: require-skill
  skill: ai-research-explore
upstream:
  repo: github.com/lllllllama/rigorpilot-skills
  path: skills/ai-research-explore/SKILL.md
  ref: fb3ccdf5aa64b1fa5b1d0463e666a8bc8bc26e01
  content_hash: sha256:e6ceb60ef42257f9b0c7c91e63e20173bf0fb6cf133df13dbe8e308911183e0e
provenance:
  stars: 486
  license: MIT
  last_push: '2026-09-23'
  fetched: '2026-11-12'
summary: Runs gated, candidate-only deep learning research exploration on a current_research anchor, logged to explore_outputs/.
tags:
- practice
- ml-research
---
Loads ai-research-explore before explore_outputs/ records are written, so candidate runs stay gated and comparable to the anchor.
