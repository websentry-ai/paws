---
name: brew
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((brew)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(install|upgrade|uninstall|reinstall|tap|untap|pin|unpin|autoremove|cleanup|link)\b)
action:
  type: require-skill
  skill: brew
  on_miss: digest
  digest: digests/brew.md
summary: 'Claude never runs brew with sudo, checks what is outdated before a bulk upgrade, installs only from approved taps, respects pins, and previews what a removal deletes.'
tags: [org, packages]
---
Loads the brew skill before Homebrew installs, upgrades, removals, taps, pins, cleanups and links.
