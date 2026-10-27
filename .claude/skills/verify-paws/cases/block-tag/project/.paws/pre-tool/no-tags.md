---
name: no-manual-tags
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*git\s+tag\b
action:
  type: block
  deny_message: Tags are cut by the release pipeline, never by hand.
---
