---
name: caveman-review
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*gh\s+pr\s+review($|\s)[^;&|\n]*\s(-b|--body|-F|--body-file)(\s|=|$)
action:
  type: require-skill
  skill: caveman-review
upstream:
  repo: github.com/juliusbrussee/caveman
  path: skills/caveman-review/SKILL.md
  ref: 3dc497a7f858c774d6e15b25e8c6120ef3a0eda8
  content_hash: sha256:c55908f8ee0dca0e59690e939dfd4a877bd9361897ae5fced096274a77509154
provenance:
  stars: 118957
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Writes code review comments as one terse line per finding: location, problem, fix.'
tags:
- practice
- writing
---
Loads caveman-review before a PR review with a comment body is posted, so each finding is one terse line.
