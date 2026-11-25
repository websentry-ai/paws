---
name: datadog-mcp
event: pre-tool
matcher:
  tool: mcp__datadog__*|mcp__plugin_datadog_datadog__*
  mcp_action: create|update|delete
action:
  type: require-skill
  skill: datadog
summary: 'Tidy Datadog changes: mutes that expire, monitors diffed before edits, and an owner tag on everything new.'
tags: [org, observability]
---
Loads the datadog skill before a Datadog MCP create, update or delete (monitors, incidents, dashboards, downtimes).
