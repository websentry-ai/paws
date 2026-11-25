---
name: "vault"
event: "pre-tool"
matcher:
  tool: "Bash"
  command_regex: "((^|[;&|\\n])\\s*|\\$\\()(\\(\\s*|(sudo\\s+)?(\\S*/)?(ba|z|da|k)?sh(\\s+-\\w+)*\\s+-\\w*c\\w*\\s+[\"']?)*\\s*(\\w+=\\S*\\s+)*(sudo(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+)?((vault)(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+(kv\\s+(get|put|patch|delete|metadata\\s+delete)|token\\s+create|policy\\s+write|lease\\s+revoke)\\b)"
action:
  type: require-skill
  skill: "vault"
---
Secrets are read, written or deleted, tokens created
