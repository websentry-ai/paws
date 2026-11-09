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
summary: 'Claude keeps Cargo.lock committed and builds with --locked, checks advisories before adding a crate, reads build.rs for unfamiliar crates, and notes global installs.'
tags: [org, dependencies]
---
Loads the cargo skill before Rust dependencies are added, installed or updated.
