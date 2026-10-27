---
name: kubectl
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((kubectl)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(apply|create|delete|edit|patch|scale|drain|cordon|uncordon|rollout|exec|label|annotate)\b|(helm)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(install|upgrade|uninstall|rollback)\b)
action:
  type: require-skill
  skill: kubectl
  on_miss: digest
  digest: digests/kubectl.md
tags: [org, kubernetes]
---
Loads the kubectl skill before any cluster mutation through kubectl or helm.
