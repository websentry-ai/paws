---
name: release-tags-digest
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*git\s+tag\b
action:
  type: require-skill
  skill: release-tags
  on_miss: digest
  digest: digests/release-tags.md
---
