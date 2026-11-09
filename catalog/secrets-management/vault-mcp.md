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
summary: 'Safe Vault use: secrets never printed or passed as arguments, rotation after a leak, and short-lived narrow tokens.'
tags: [org, secrets]
---
Loads the vault skill before a Vault MCP secret write, delete or token create.
