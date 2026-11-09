---
name: datadog-mcp
event: pre-tool
matcher:
  tool: mcp__datadog__*|mcp__plugin_datadog_datadog__*
  mcp_action: create|update|delete
action:
  type: require-skill
  skill: datadog
  on_miss: digest
  digest: digests/datadog.md
summary: 'Claude gives every Datadog mute an end time, fetches and diffs a monitor before editing it, tags new monitors with a team and owner, and confirms before deleting shared ones.'
tags: [org, observability]
---
Loads the datadog skill before a Datadog MCP create, update or delete (monitors, incidents, dashboards, downtimes).
