---
name: npm-deps
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((npm)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(install|i|add|ci|update|up|rebuild)\b|npx\b)
action:
  type: require-skill
  skill: npm-deps
  on_miss: digest
  digest: digests/npm-deps.md
tags: [org, dependencies]
---
Loads the npm-deps skill before npm installs, updates, rebuilds and npx runs.
