---
name: terraform-mcp
event: pre-tool
matcher:
  tool: mcp__terraform__create_run|mcp__terraform__apply_run
action:
  type: require-skill
  skill: terraform
  on_miss: digest
  digest: digests/terraform.md
summary: 'Safe Terraform: apply only a reviewed saved plan, back up state before editing it, and confirm before destroy.'
tags: [org, iac]
---
Loads the terraform skill before a Terraform Cloud MCP run is created or applied.
