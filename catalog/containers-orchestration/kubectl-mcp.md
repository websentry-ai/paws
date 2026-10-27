---
name: kubectl-mcp
event: pre-tool
matcher:
  tool: mcp__kubernetes__*|mcp__k8s__*
  mcp_action: create|update|delete
action:
  type: require-skill
  skill: kubectl
  on_miss: digest
  digest: digests/kubectl.md
tags: [org, kubernetes]
---
Loads the kubectl skill before a Kubernetes MCP server creates, applies, patches or deletes resources.
