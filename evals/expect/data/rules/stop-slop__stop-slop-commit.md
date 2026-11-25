---
name: "stop-slop-commit"
event: "pre-commit"
matcher:
  tool: "Bash"
action:
  type: require-skill
  skill: "stop-slop"
---
A commit message is written
