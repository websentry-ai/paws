---
name: caveman-learn
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?caveman\s+(learn|mem)\b
action:
  type: require-skill
  skill: caveman-learn
upstream:
  repo: github.com/juliusbrussee/caveman
  path: skills/caveman-learn/SKILL.md
  ref: 3dc497a7f858c774d6e15b25e8c6120ef3a0eda8
  content_hash: sha256:bd8f0bedab21b580cddee401846aa81723378334548f41863fc131c58d612556
provenance:
  stars: 118957
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Reviews the caveman learn token-sink report and applies cost-cutting edits one at a time with consent.
tags:
- practice
- observability
---
Loads caveman-learn before the learn report or cavemem is used, so edits are made one at a time with consent.
