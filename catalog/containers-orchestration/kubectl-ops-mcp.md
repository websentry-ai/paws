---
name: kubectl-ops-mcp
event: pre-tool
matcher:
  tool: mcp__kubernetes__kubectl_scale|mcp__kubernetes__kubectl_rollout|mcp__kubernetes__pods_exec|mcp__kubernetes__pods_run|mcp__kubernetes__exec_in_pod|mcp__kubernetes__scale_deployment|mcp__kubernetes__install_helm_chart|mcp__kubernetes__upgrade_helm_chart|mcp__kubernetes__uninstall_helm_chart|mcp__kubernetes__helm_install|mcp__kubernetes__helm_uninstall|mcp__k8s__kubectl_scale|mcp__k8s__kubectl_rollout|mcp__k8s__pods_exec|mcp__k8s__pods_run|mcp__k8s__exec_in_pod|mcp__k8s__scale_deployment|mcp__k8s__install_helm_chart|mcp__k8s__upgrade_helm_chart|mcp__k8s__uninstall_helm_chart|mcp__k8s__helm_install|mcp__k8s__helm_uninstall
action:
  type: require-skill
  skill: kubectl
  on_miss: digest
  digest: digests/kubectl.md
tags: [org, kubernetes]
---
Loads the kubectl skill before Kubernetes MCP scale, rollout, exec, run and helm operations, whose tool names carry no create/update/delete verb.
