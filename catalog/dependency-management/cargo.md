---
name: cargo
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((cargo)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(add|install|update)\b)
action:
  type: require-skill
  skill: cargo
  on_miss: digest
  digest: digests/cargo.md
tags: [org, dependencies]
---
Loads the cargo skill before Rust dependencies are added, installed or updated.
