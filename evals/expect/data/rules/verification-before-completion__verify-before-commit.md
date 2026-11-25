---
name: verify-before-commit
event: pre-commit
matcher:
  tool: "Bash"
action:
  type: require-skill
  skill: verification-before-completion
---
A commit
