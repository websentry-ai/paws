---
name: "terraform"
event: "pre-tool"
matcher:
  tool: "Bash"
  command_regex: "((^|[;&|\\n])\\s*|\\$\\()(\\(\\s*|(sudo\\s+)?(\\S*/)?(ba|z|da|k)?sh(\\s+-\\w+)*\\s+-\\w*c\\w*\\s+[\"']?)*\\s*(\\w+=\\S*\\s+)*(sudo(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+)?((terraform|tofu)(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+(apply|destroy|state\\s+(rm|mv|push)|import|force-unlock|taint|untaint|workspace\\s+delete)\\b)"
action:
  type: require-skill
  skill: "terraform"
---
Apply, destroy, state surgery, import, unlock, taint and workspace deletes
