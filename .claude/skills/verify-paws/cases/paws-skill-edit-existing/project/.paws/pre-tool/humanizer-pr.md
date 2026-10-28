---
name: humanizer-pr
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*gh\s+pr\s+create\b
action:
  type: require-skill
  skill: humanizer
---
PR bodies read like a person wrote them.
