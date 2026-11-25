---
name: "postgres"
event: "pre-tool"
matcher:
  tool: "Bash"
  command_regex: "((^|[;&|\\n])\\s*|\\$\\()(\\(\\s*|(sudo\\s+)?(\\S*/)?(ba|z|da|k)?sh(\\s+-\\w+)*\\s+-\\w*c\\w*\\s+[\"']?)*\\s*(\\w+=\\S*\\s+)*(sudo(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+)?(psql\\s[^|&\\n]*(?i:\\b(update|delete|insert|drop|truncate|alter|create)\\b)|psql\\s[^|&\\n]*\\bprod(uction)?\\b|pg_restore\\b|pg_dump(all)?\\b)"
action:
  type: require-skill
  skill: "postgres"
---
Psql runs write SQL or opens a production session
