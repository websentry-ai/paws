---
name: using-git-worktrees
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?git\s+(-C\s+\S+\s+)?worktree\s+add\b
action:
  type: require-skill
  skill: using-git-worktrees
upstream:
  repo: github.com/obra/superpowers
  path: skills/using-git-worktrees/SKILL.md
  ref: 22f0b463e7a5e579904dd6211a520321035526f7
  content_hash: sha256:b69324cb5e5ba5b846fc2ae2811616008dc6f19858245100e472713190d4c9e4
provenance:
  stars: 315870
  license: MIT
  last_push: '2026-11-03'
  fetched: '2026-11-12'
tags:
- practice
- git
---
Loads using-git-worktrees before a worktree is created or entered, so existing isolation is detected and the native tool is preferred.
