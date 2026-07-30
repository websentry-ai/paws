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
# Block Production Secrets

Denies any Read/Write/Edit, or Bash command, whose target looks like a
production secrets file (`.env`, `secrets/`, `prod.env`). Regex-based, so
determined obfuscation (e.g. base64, indirection) can still slip through —
it stops the common case, not a determined attacker.
