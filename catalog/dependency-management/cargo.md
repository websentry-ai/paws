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
summary: 'Safe Rust dependencies: a committed Cargo.lock, an advisory check before adding a crate, and build.rs read first.'
tags: [org, dependencies]
---
Loads the cargo skill before Rust dependencies are added, installed or updated.
