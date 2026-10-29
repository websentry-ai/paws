---
name: apt
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((apt|apt-get)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(install|upgrade|dist-upgrade|remove|purge|autoremove)\b|add-apt-repository\b|dpkg(\s+-\S+)*\s+(-i|--install)\b)
action:
  type: require-skill
  skill: apt
  on_miss: digest
  digest: digests/apt.md
tags: [org, packages]
---
Loads the apt skill before packages are installed, upgraded or removed with apt, apt-get or dpkg, and before apt sources are added.
