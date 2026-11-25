---
name: "rclone"
event: "pre-tool"
matcher:
  tool: "Bash"
  command_regex: "((^|[;&|\\n])\\s*|\\$\\()(\\(\\s*|(sudo\\s+)?(\\S*/)?(ba|z|da|k)?sh(\\s+-\\w+)*\\s+-\\w*c\\w*\\s+[\"']?)*\\s*(\\w+=\\S*\\s+)*(sudo(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+)?((rclone)(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+(copy(to)?|sync|move(to)?|delete(file)?|purge)\\b|(rclone)(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+config(\\s+(create|update|delete|password|edit|reconnect|disconnect)\\b|\\s*($|[;&|\\n])))"
action:
  type: require-skill
  skill: "rclone"
---
Bulk copies, syncs, moves and deletes, and before remotes are configured
