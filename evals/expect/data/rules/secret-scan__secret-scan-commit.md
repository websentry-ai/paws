---
name: secret-scan-commit
event: pre-commit
matcher:
  tool: "Bash"
action:
  type: require-skill
  skill: secret-scan
---
A commit
