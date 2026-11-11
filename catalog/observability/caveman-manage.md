---
name: caveman-manage
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?caveman\s+cloud\s+experiments?\b
action:
  type: require-skill
  skill: caveman-manage
upstream:
  repo: github.com/juliusbrussee/caveman
  path: skills/caveman-manage/SKILL.md
  ref: 3dc497a7f858c774d6e15b25e8c6120ef3a0eda8
  content_hash: sha256:2dd0ef43e4a5412913967513a494c2db2219fa34630378730409205f3a0b883a
provenance:
  stars: 118957
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Inspects Caveman Cloud experiment state and blocks unsafe approvals, promotions or rollbacks.
tags:
- practice
- observability
---
Loads caveman-manage before Caveman Cloud experiments are inspected or changed, so unsafe approvals are blocked.
