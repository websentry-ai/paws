---
name: commit-crew-roster
event: pre-commit
matcher:
  tool: Bash
action:
  type: require-skill
  skill: crew-roster
---
Commits follow the crew-roster conventions.
