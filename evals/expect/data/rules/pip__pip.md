---
name: "pip"
event: "pre-tool"
matcher:
  tool: "Bash"
  command_regex: "((^|[;&|\\n])\\s*|\\$\\()(\\(\\s*|(sudo\\s+)?(\\S*/)?(ba|z|da|k)?sh(\\s+-\\w+)*\\s+-\\w*c\\w*\\s+[\"']?)*\\s*(\\w+=\\S*\\s+)*(sudo(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+)?((pip3?)(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+(install|download)\\b|(python3?\\s+-m\\s+pip)(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+(install|download)\\b|(uv)(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+(add|pip\\s+install)\\b)"
action:
  type: require-skill
  skill: "pip"
---
Python packages are installed or downloaded with pip or uv
