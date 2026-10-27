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
tags: [org, network]
---
Loads the rsync skill before an rsync that propagates deletes or crosses to a remote host.
