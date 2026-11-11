---
name: receiving-code-review
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?gh\s+(pr\s+view\b[^;&|\n]*--comments|api\s+["\']?repos/\S+/pulls/\d+/(comments|reviews)["\']?(\s+--paginate)?(\s+(--jq|-q)\s+(\'[^\']*\'|"[^"]*"|\S+))*\s*($|[;&|\n)]))
action:
  type: require-skill
  skill: receiving-code-review
upstream:
  repo: github.com/obra/superpowers
  path: skills/receiving-code-review/SKILL.md
  ref: 22f0b463e7a5e579904dd6211a520321035526f7
  content_hash: sha256:528d336858d6f7d22cfcb887a249409ce682cb7d37a939cb15a82f2576a35876
provenance:
  stars: 315870
  license: MIT
  last_push: '2026-11-03'
  fetched: '2026-11-12'
tags:
- practice
- source-control
---
Loads receiving-code-review before PR review comments are fetched, so feedback is checked against the code before anything is implemented.
