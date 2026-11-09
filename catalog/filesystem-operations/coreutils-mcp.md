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
summary: 'Before rm, dd, chmod -R or unpacking an archive, Claude lists what will be touched, echoes resolved variables, reads back the target disk, and looks inside the archive.'
tags: [org, filesystem]
---
Loads the coreutils skill before the filesystem MCP server moves or overwrites a file.
