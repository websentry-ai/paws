---
name: rclone
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((rclone)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(copy(to)?|sync|move(to)?|delete(file)?|purge)\b|(rclone)(\s+-\S+(\s+[^-\s]\S*)?)*\s+config(\s+(create|update|delete|password|edit|reconnect|disconnect)\b|\s*($|[;&|\n])))
action:
  type: require-skill
  skill: rclone
  on_miss: digest
  digest: digests/rclone.md
tags: [org, cloud]
---
Loads the rclone skill before bulk copies, syncs, moves and deletes, and before remotes are configured.
