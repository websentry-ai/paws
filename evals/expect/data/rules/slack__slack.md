---
name: "slack"
event: "pre-tool"
matcher:
  tool: "Bash"
  command_regex: "((^|[;&|\\n])\\s*|\\$\\()(\\(\\s*|(sudo\\s+)?(\\S*/)?(ba|z|da|k)?sh(\\s+-\\w+)*\\s+-\\w*c\\w*\\s+[\"']?)*\\s*(\\w+=\\S*\\s+)*(sudo(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+)?curl\\s[^;&|\\n]*slack\\.com/api/(chat\\.(postMessage|scheduleMessage|update|delete)|files\\.upload)"
action:
  type: require-skill
  skill: "slack"
---
A message is posted, scheduled, edited
