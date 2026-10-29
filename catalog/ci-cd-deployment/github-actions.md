---
name: github-actions
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?(gh)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(workflow\s+(run|enable|disable)\b|release\s+(create|delete)\b|run\s+cancel\b|pr\s+merge\b[^;&|\n]*\s--admin\b)
action:
  type: require-skill
  skill: github-actions
  on_miss: digest
  digest: digests/github-actions.md
tags: [org, deploy]
---
Loads the github-actions skill before workflow dispatches, release create/delete, run cancels and admin-bypass merges.
