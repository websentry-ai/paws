---
name: apt
event: pre-tool
matcher:
  tool: Bash
  command_regex: ((^|[;&|\n])\s*|\$\()(\(\s*|(sudo\s+)?(\S*/)?(ba|z|da|k)?sh(\s+-\w+)*\s+-\w*c\w*\s+["']?)*\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((apt|apt-get)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(install|upgrade|dist-upgrade|remove|purge|autoremove)\b|add-apt-repository\b|dpkg(\s+-\S+)*\s+(-i|--install)\b)
action:
  type: require-skill
  skill: apt
summary: 'Safe apt habits: refresh before installing, simulate big upgrades and removals, and add only key-verified sources.'
tags: [org, packages]
---
Loads the apt skill before packages are installed, upgraded or removed with apt, apt-get or dpkg, and before apt sources are added.
