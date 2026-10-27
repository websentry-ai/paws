---
name: watch-status
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*git\s+status\b
action:
  type: audit
---
