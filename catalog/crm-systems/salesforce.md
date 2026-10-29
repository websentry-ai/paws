---
name: salesforce
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((sf)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(data\s+((create|update|delete)\s+record|upsert|import|delete\s+bulk)|apex\s+run)\b)
action:
  type: require-skill
  skill: salesforce
  on_miss: digest
  digest: digests/salesforce.md
tags: [org, crm]
---
Loads the salesforce skill before Salesforce records are created, updated, upserted, imported or deleted, or Apex runs.
