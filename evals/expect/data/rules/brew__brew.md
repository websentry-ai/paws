---
name: brew
event: pre-tool
matcher:
  tool: Bash
  command_regex: ((^|[;&|\n])\s*|\$\()(\(\s*|(sudo\s+)?(\S*/)?(ba|z|da|k)?sh(\s+-\w+)*\s+-\w*c\w*\s+["']?)*\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((brew)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(install|upgrade|uninstall|reinstall|tap|untap|pin|unpin|autoremove|cleanup|link)\b)
action:
  type: require-skill
  skill: brew
summary: 'Safe Homebrew habits: never sudo, check what is outdated before bulk upgrades, and preview what a removal deletes.'
tags: [org, packages]
---
Loads the brew skill before Homebrew installs, upgrades, removals, taps, pins, cleanups and links.
