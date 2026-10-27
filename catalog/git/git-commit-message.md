---
name: git-commit-message
event: pre-commit
matcher:
  tool: Bash
action:
  type: require-skill
  skill: git-commit
upstream:
  repo: github.com/github/awesome-copilot
  path: skills/git-commit/SKILL.md
  ref: d75b41863510272ed217a3074c7045202b96e93f
  content_hash: sha256:92a8786484fe7854e1a1b875fafd71c8795b2e3cffd08d968b31b91db17f056c
provenance:
  stars: 40468
  license: MIT
  last_push: 2026-10-27
  fetched: 2026-10-27
tags: [practice, git]
---
Loads git-commit before a commit, so the message follows Conventional Commits and the staged files belong together.
