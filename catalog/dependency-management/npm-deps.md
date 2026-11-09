---
name: npm-deps
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((npm)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(install|i|add)(\s+-\S+)*\s+[^-\s;&|]|(yarn|pnpm|bun)(\s+-\S+(\s+[^-\s]\S*)?)*\s+add\b)
action:
  type: require-skill
  skill: npm-deps
  on_miss: digest
  digest: digests/npm-deps.md
summary: 'Claude uses npm ci in existing projects, checks a package before adding it, installs unfamiliar packages with --ignore-scripts first, and never uses sudo.'
tags: [org, dependencies]
---
Loads the npm-deps skill before a new package is added: npm install/i/add <pkg>, yarn add, pnpm add, bun add.
