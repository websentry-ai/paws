---
name: pr
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?gh\s+pr\s+(create|edit)\b[^;&|\n]*\s(--body(-file)?|-b|-F)([=\s]|$)
action:
  type: require-skill
  skill: pr
upstream:
  repo: github.com/mattpocock/skills
  path: skills/engineering/pr/SKILL.md
  ref: 4ae2317ddb22b046e26455e4f9c9b3aaaf62212d
  content_hash: sha256:a66195583d0f9323004cdd4da169f93449071f29023671f3e31a916e740f6861
provenance:
  stars: 321219
  license: MIT
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Writes a pull request body in a set structure, with visuals when they help.
tags:
- practice
- source-control
---
Loads pr before a pull request body is supplied to gh, so the Summary, Evidence and Merge Danger structure is used.
