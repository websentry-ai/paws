---
name: terraform
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((terraform|tofu)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(apply|destroy|state\s+(rm|mv|push)|import|force-unlock|taint|untaint|workspace\s+delete)\b)
action:
  type: require-skill
  skill: terraform
  on_miss: digest
  digest: digests/terraform.md
tags: [org, iac]
---
Loads the terraform skill before apply, destroy, state surgery, import, unlock, taint and workspace deletes (Terraform and OpenTofu).
