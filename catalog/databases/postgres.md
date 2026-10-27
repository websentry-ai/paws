---
name: postgres
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?(psql\s[^|&\n]*(?i:\b(update|delete|insert|drop|truncate|alter|create)\b)|psql\s[^|&\n]*\bprod(uction)?\b|pg_restore\b|pg_dump(all)?\b)
action:
  type: require-skill
  skill: postgres
  on_miss: digest
  digest: digests/postgres.md
tags: [org, database]
---
Loads the postgres skill before psql runs write SQL or opens a production session, and before pg_dump or pg_restore.
