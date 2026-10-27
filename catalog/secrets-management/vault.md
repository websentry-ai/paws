---
name: vault
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((vault)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(kv\s+(get|put|patch|delete|metadata\s+delete)|token\s+create|policy\s+write|lease\s+revoke)\b)
action:
  type: require-skill
  skill: vault
  on_miss: digest
  digest: digests/vault.md
tags: [org, secrets]
---
Loads the vault skill before secrets are read, written or deleted, tokens created, policies written or leases revoked.
