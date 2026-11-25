---
name: "stop-slop-pr"
event: "pre-tool"
matcher:
  tool: "Bash"
  command_regex: "((^|[;&|\\n])\\s*|\\$\\()(\\(\\s*|(sudo\\s+)?(\\S*/)?(ba|z|da|k)?sh(\\s+-\\w+)*\\s+-\\w*c\\w*\\s+[\"']?)*\\s*gh\\s+pr\\s+(create|edit)\\b"
action:
  type: require-skill
  skill: "stop-slop"
---
A PR title or body goes out
