---
name: bigquery
event: pre-tool
matcher:
  tool: Bash
  command_regex: ((^|[;&|\n])\s*|\$\()(\(\s*|(sudo\s+)?(\S*/)?(ba|z|da|k)?sh(\s+-\w+)*\s+-\w*c\w*\s+["']?)*\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((bq)(\s+-\S+(\s+[^-\s]\S*)?)*\s+query\b(\s+([^-\s;&|][^\s;&|]*|-[^-\s;&|][^\s;&|]*|--|--([^d\s;&|]|d[^r])[^\s;&|]*|--dry.run=(server|none)[^\s;&|]*))*\s*($|[;&|)\n])|(bq)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(mk|rm|cp|load|extract|update)\b|(dbt)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(run|build)\b)
action:
  type: require-skill
  skill: bigquery
tags: [org, data]
---
Loads the bigquery skill before queries (cost preview first), dataset/table writes and dbt runs.
