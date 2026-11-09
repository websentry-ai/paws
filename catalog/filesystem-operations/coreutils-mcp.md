---
name: coreutils-mcp
event: pre-tool
matcher:
  tool: mcp__filesystem__move_file|mcp__filesystem__write_file
action:
  type: require-skill
  skill: coreutils
  on_miss: digest
  digest: digests/coreutils.md
summary: 'Safe destructive shell commands: list what will be deleted, check variables before rm, and look inside archives first.'
tags: [org, filesystem]
---
Loads the coreutils skill before the filesystem MCP server moves or overwrites a file.
