---
name: "npm"
event: "pre-tool"
matcher:
  tool: "Bash"
  command_regex: "((^|[;&|\\n])\\s*|\\$\\()(\\(\\s*|(sudo\\s+)?(\\S*/)?(ba|z|da|k)?sh(\\s+-\\w+)*\\s+-\\w*c\\w*\\s+[\"']?)*\\s*(\\w+=\\S*\\s+)*(sudo(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+)?((npm)(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+(publish|unpublish|deprecate|dist-tag\\s+(add|rm|set)|access\\s+(public|restricted|grant|revoke|set)|token\\s+(create|revoke|rm|delete))\\b|(yarn|pnpm)(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+(publish)\\b)"
action:
  type: require-skill
  skill: "npm"
---
Packages are published, unpublished, deprecated or re-tagged
