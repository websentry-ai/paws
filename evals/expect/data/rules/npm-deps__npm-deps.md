---
name: "npm-deps"
event: "pre-tool"
matcher:
  tool: "Bash"
  command_regex: "((^|[;&|\\n])\\s*|\\$\\()(\\(\\s*|(sudo\\s+)?(\\S*/)?(ba|z|da|k)?sh(\\s+-\\w+)*\\s+-\\w*c\\w*\\s+[\"']?)*\\s*(\\w+=\\S*\\s+)*(sudo(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+)?((npm)(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+(install|i|add)(\\s+-\\S+)*\\s+[^-\\s;&|]|(yarn|pnpm|bun)(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+add\\b)"
action:
  type: require-skill
  skill: "npm-deps"
---
A new package is added: npm install/i/add <pkg>, yarn add, pnpm add, bun add
