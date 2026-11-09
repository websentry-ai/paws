---
name: rsync
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?(rsync\s[^;&|\n]*(--del|\S:))
action:
  type: require-skill
  skill: rsync
  on_miss: digest
  digest: digests/rsync.md
summary: 'Safe rsync habits: preview anything that deletes, get the trailing slash right, and keep a backup when a sync removes files.'
tags: [org, network]
---
Loads the rsync skill before an rsync that propagates deletes or crosses to a remote host.
