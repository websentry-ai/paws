---
name: "vault-mcp"
event: "pre-tool"
matcher:
  tool: "mcp__vault__write_secret|mcp__vault__delete_secret|mcp__vault__create_token"
action:
  type: require-skill
  skill: "vault"
---
A Vault MCP secret write, delete or token create
