---
name: attio-mcp
event: pre-tool
matcher:
  tool: mcp__attio__create-record|mcp__attio__update-record|mcp__attio__upsert-record|mcp__attio__create-note|mcp__attio__update-note|mcp__attio__create-task|mcp__attio__add-record-to-list|mcp__attio__update-list-entry-by-record-id
action:
  type: require-skill
  skill: attio
  on_miss: digest
  digest: digests/attio.md
summary: 'Claude searches Attio for an existing record before creating one, reads the attribute schema before its first write, uses select values exactly, and signs its notes.'
tags: [org, crm]
---
Loads the attio skill before CRM records, notes, tasks or list entries are written in Attio.
