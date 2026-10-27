---
name: commit-house-style
event: pre-commit
matcher:
  tool: Bash
action:
  type: require-skill
  skill: house-style
---
Commits must follow the house-style skill.
