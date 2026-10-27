---
name: npm
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((npm)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(publish|unpublish|deprecate|version|dist-tag|access|token)\b|(yarn|pnpm)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(publish)\b)
action:
  type: require-skill
  skill: npm
  on_miss: digest
  digest: digests/npm.md
tags: [org, publishing]
---
Loads the npm skill before packages are published, unpublished, deprecated, versioned, re-tagged, or access and tokens change.
