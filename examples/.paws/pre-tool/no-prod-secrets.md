---
name: no-prod-secrets
event: pre-tool
matcher:
  tool: Read|Write|Edit|Bash
  command_regex: (\.env(\.|$)|secrets/|prod\.env)
action:
  type: block
  deny_message: paws blocks reading or writing production secret files via file tools or the shell.
---
# no-prod-secrets
Blocks reading/writing files that look like prod secrets.
