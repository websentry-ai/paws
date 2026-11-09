---
name: launchctl
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((launchctl)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(bootstrap|bootout|load|unload|enable|disable|kickstart|kill)\b)
action:
  type: require-skill
  skill: launchctl
  on_miss: digest
  digest: digests/launchctl.md
summary: 'Careful launchd changes on macOS: read existing jobs first, review the full plist, and flag anything that persists.'
tags: [org, system]
---
Loads the launchctl skill before launchd jobs are loaded, unloaded, enabled, disabled, kicked or killed.
