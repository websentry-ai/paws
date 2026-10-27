---
name: datadog
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?(curl\s[^;&|\n]*api\.([a-z0-9]+\.)?(datadoghq\.(com|eu)|ddog-gov\.com)/api/v1/monitor/[^/\s]+/(un)?mute|curl\s[^;&|\n]*(-X\s*|--request\s+)(POST|PUT|PATCH|DELETE)\b[^;&|\n]*api\.([a-z0-9]+\.)?(datadoghq\.(com|eu)|ddog-gov\.com)/api/(v1/(monitor|dashboard|downtime)|v2/incidents)|curl\s[^;&|\n]*api\.([a-z0-9]+\.)?(datadoghq\.(com|eu)|ddog-gov\.com)/api/(v1/(monitor|dashboard|downtime)|v2/incidents)[^;&|\n]*(-X\s*|--request\s+)(POST|PUT|PATCH|DELETE)\b|(dog)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(monitor\s+(mute|unmute|update|delete)|dashboard)\b)
action:
  type: require-skill
  skill: datadog
  on_miss: digest
  digest: digests/datadog.md
tags: [org, observability]
---
Loads the datadog skill before monitors are muted, changed or deleted, dashboards or downtimes written, and incidents changed.
