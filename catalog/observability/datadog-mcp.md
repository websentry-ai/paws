---
name: datadog-mcp
event: pre-tool
matcher:
  tool: mcp__datadog__mute_monitor|mcp__datadog__update_monitor|mcp__datadog__create_monitor|mcp__datadog__delete_monitor|mcp__datadog__update_incident
action:
  type: require-skill
  skill: datadog
  on_miss: digest
  digest: digests/datadog.md
tags: [org, observability]
---
Loads the datadog skill before a Datadog MCP monitor or incident write.
