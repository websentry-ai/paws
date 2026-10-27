---
name: kubectl-mcp
event: pre-tool
matcher:
  tool: mcp__kubernetes__apply_manifest|mcp__kubernetes__delete_resource|mcp__kubernetes__exec_in_pod|mcp__kubernetes__scale_deployment
action:
  type: require-skill
  skill: kubectl
  on_miss: digest
  digest: digests/kubectl.md
tags: [org, kubernetes]
---
Loads the kubectl skill before a Kubernetes MCP server applies, deletes, scales or execs.
