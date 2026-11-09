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
summary: 'Claude passes --context and -n on every kubectl write, runs a server-side dry-run before prod applies, checks images are pinned and limits set, and states the blast radius first.'
tags: [org, kubernetes]
---
Loads the kubectl skill before a Kubernetes MCP server creates, applies, patches or deletes resources.
