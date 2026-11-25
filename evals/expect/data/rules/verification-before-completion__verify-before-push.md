---
name: verify-before-push
event: pre-push
matcher:
  tool: "Bash"
action:
  type: require-skill
  skill: verification-before-completion
---
A push
