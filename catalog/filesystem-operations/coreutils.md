---
name: coreutils
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?(rm(\s+-\S+)*\s+(-[a-zA-Z]*[rRf]|--recursive\b|--force\b)|find(\s+[^\s;&|]+)*\s+(-delete\b|-exec(dir)?\s+rm\b)|dd(\s+[^\s;&|]+)*\s+of=/dev/|mkfs\b|diskutil\s+(erase\w*|partitionDisk)\b|(chmod|chown)(\s+-\S+)*\s+(-[a-zA-Z]*R[a-zA-Z]*|--recursive)\b|chmod(\s+[^\s;&|]+)*\s+[^\s;&|]*(777|\+s)|shred\b|tar(\s+[^\s;&|]+)*\s+(--absolute-names\b|-[a-zA-Z]*P))
action:
  type: require-skill
  skill: coreutils
  on_miss: digest
  digest: digests/coreutils.md
tags: [org, filesystem]
---
Loads the coreutils skill before recursive or forced deletes, find -delete, raw device writes, disk erases, recursive or setuid permission changes, shred and absolute-path tar.
