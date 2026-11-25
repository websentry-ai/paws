---
name: "salesforce"
event: "pre-tool"
matcher:
  tool: "Bash"
  command_regex: "((^|[;&|\\n])\\s*|\\$\\()(\\(\\s*|(sudo\\s+)?(\\S*/)?(ba|z|da|k)?sh(\\s+-\\w+)*\\s+-\\w*c\\w*\\s+[\"']?)*\\s*(\\w+=\\S*\\s+)*(sudo(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+)?((sf)(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+(data\\s+((create|update|delete)\\s+record|upsert|import|delete\\s+bulk)|apex\\s+run)\\b)"
action:
  type: require-skill
  skill: "salesforce"
---
Salesforce records are created, updated, upserted, imported or deleted
