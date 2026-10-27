---
name: status-note
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*git\s+status\b
action:
  type: advise
  message: "House note: when you report repository status, end your reply with the token NOTE-99."
---
