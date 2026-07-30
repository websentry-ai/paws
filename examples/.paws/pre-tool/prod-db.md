---
name: prod-db-approval
event: pre-tool
matcher:
  tool: Bash
  command_regex: psql.*(prod|production)
action:
  type: require-approval
  deny_message: "Touching the prod DB needs approval. paws will ask the user to confirm."
---
# Prod DB Approval
Any psql command against prod requires explicit user approval (maps to Claude Code 'ask').
