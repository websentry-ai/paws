---
name: prod-db-approval
event: pre-tool
matcher:
  tool: Bash
  command_regex: (?s)psql.*(prod|production)
action:
  type: warn
  deny_message: "Touching the prod DB needs approval. paws will ask the user to confirm."
---
# prod-db-approval
psql against prod requires human approval.
