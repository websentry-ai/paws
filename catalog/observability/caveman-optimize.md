---
name: caveman-optimize
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?caveman\s+opportunities\b
action:
  type: require-skill
  skill: caveman-optimize
upstream:
  repo: github.com/juliusbrussee/caveman
  path: skills/caveman-optimize/SKILL.md
  ref: 3dc497a7f858c774d6e15b25e8c6120ef3a0eda8
  content_hash: sha256:dc5eb46a2918c7250a069b1bcda3c5f385715801d42c317c251ba5208ccd685e
provenance:
  stars: 118957
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Turns a Caveman optimization observation into an operator-chosen candidate with a paired baseline evaluation.
tags:
- practice
- observability
---
Loads caveman-optimize before the opportunities list is read, so the operator chooses the candidate and nothing is ranked or promised.
