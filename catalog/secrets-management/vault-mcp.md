---
name: vault-mcp
event: pre-tool
matcher:
  tool: mcp__vault__write_secret|mcp__vault__delete_secret|mcp__vault__create_token
action:
  type: require-skill
  skill: vault
  on_miss: digest
  digest: digests/vault.md
summary: 'Claude reads Vault secrets with -field into a pipe so they never print, writes them from stdin, rotates any that leak, and creates narrow, short-lived tokens.'
tags: [org, secrets]
---
Loads the vault skill before a Vault MCP secret write, delete or token create.
