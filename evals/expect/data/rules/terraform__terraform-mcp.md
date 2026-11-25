---
name: "terraform-mcp"
event: "pre-tool"
matcher:
  tool: "mcp__terraform__create_run|mcp__terraform__apply_run"
action:
  type: require-skill
  skill: "terraform"
---
A Terraform Cloud MCP run is created or applied
