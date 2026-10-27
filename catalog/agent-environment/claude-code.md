---
name: claude-code
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((claude)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(mcp\s+(add|remove)|config\s+set|plugin\s+(install|i|uninstall|remove|enable|disable|update|marketplace\s+(add|remove|rm|update)))\b)
action:
  type: require-skill
  skill: claude-code
  on_miss: digest
  digest: digests/claude-code.md
tags: [org, agent]
---
Loads the claude-code skill before an agent adds or removes MCP servers, changes Claude Code config, or installs plugins.
