---
name: bigquery
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((bq)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(query|mk|rm|cp|load|extract|update)\b|(dbt)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(run|build)\b)
action:
  type: require-skill
  skill: bigquery
  on_miss: digest
  digest: digests/bigquery.md
tags: [org, data]
---
Loads the bigquery skill before queries (cost preview first), dataset/table writes and dbt runs.
