---
name: caveman-evidence-review
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?caveman\s+cloud\s+(costs|score|plan|traces|workflows|savings|whoami|projects)\b
action:
  type: require-skill
  skill: caveman-evidence-review
upstream:
  repo: github.com/juliusbrussee/caveman
  path: skills/caveman-evidence-review/SKILL.md
  ref: 3dc497a7f858c774d6e15b25e8c6120ef3a0eda8
  content_hash: sha256:cd258392c713b9cf77d2dd92f1a407a7dccaa54ced28c92192e67d76a0cfe9c6
provenance:
  stars: 118957
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Reviews Caveman Cloud cost, score, traces, latency and savings read-only, keeping measured and inferred figures apart.
tags:
- practice
- observability
---
Loads caveman-evidence-review before Caveman Cloud evidence is read, so measured and inferred figures stay separate.
