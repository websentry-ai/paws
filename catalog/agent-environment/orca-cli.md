---
name: orca-cli
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(orca|orca-ide|orca-dev)\s+[a-z]
action:
  type: require-skill
  skill: orca-cli
upstream:
  repo: github.com/stablyai/orca
  path: skills/orca-cli/SKILL.md
  ref: b0c431c1a3789b10ee96ad26eeb91b4da9ea00c4
  content_hash: sha256:3bff90cd25067ddc874ff68130f2efb762de9b2ab106f3907c846c3212d08b42
provenance:
  stars: 121731
  license: MIT
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: Drives Orca worktrees, terminals, repos, automations and the embedded browser through the version-matched orca CLI.
tags:
- practice
- agent-environment
---
Loads orca-cli before an orca CLI command runs, so the version-matched guide is read first and the right executable is used.
