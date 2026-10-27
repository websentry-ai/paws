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
tags: [org, iac]
---
Loads the terraform skill before a Terraform Cloud MCP run is created or applied.
