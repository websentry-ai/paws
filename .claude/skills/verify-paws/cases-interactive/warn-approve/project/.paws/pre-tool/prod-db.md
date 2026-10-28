---
name: prod-db-approval
event: pre-tool
matcher:
  tool: Bash
  command_regex: (?s)psql.*(prod|production)
action:
  type: warn
  deny_message: "WRN-4F: touching the prod database needs a human to approve."
---
psql against prod asks the human first.
