---
name: curl
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?(curl(\s+[^\s;&|]+)*\s+(--insecure\b|-[a-zA-Z]*k[a-zA-Z]*(\s|$)|(-X\s*|--request\s+)(POST|PUT|PATCH|DELETE)\b|-d\b|--data|-F\b|--form\b|-u\b|--user\b)|curl\s[^;&|\n]*\|\s*(sudo\s+)?(ba)?sh\b|wget(\s+[^\s;&|]+)*\s+(--no-check-certificate|--post-data))
action:
  type: require-skill
  skill: curl
  on_miss: digest
  digest: digests/curl.md
tags: [org, network]
---
Loads the curl skill before curl or wget sends credentials, disables TLS checks, writes to an API, or pipes a download into a shell.
