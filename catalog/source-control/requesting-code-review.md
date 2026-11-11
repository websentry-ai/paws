---
name: requesting-code-review
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?gh\s+pr\s+merge\b
action:
  type: require-skill
  skill: requesting-code-review
upstream:
  repo: github.com/obra/superpowers
  path: skills/requesting-code-review/SKILL.md
  ref: 22f0b463e7a5e579904dd6211a520321035526f7
  content_hash: sha256:32bcc25e8c3315d0150f646ee0067c9d4bfbc7719c34012db9c202e17cce8a14
provenance:
  stars: 315870
  license: MIT
  last_push: '2026-11-03'
  fetched: '2026-11-12'
summary: Dispatches a reviewer subagent with base and head SHAs and requirements to check work before merging.
tags:
- practice
- source-control
---
Loads requesting-code-review before gh pr merge runs, so a fresh-context reviewer checks the work first.
