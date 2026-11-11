---
name: firebase-crashlytics
event: pre-tool
matcher:
  tool: mcp__*__crashlytics_*
action:
  type: require-skill
  skill: firebase-crashlytics
upstream:
  repo: github.com/firebase/agent-skills
  path: skills/firebase-crashlytics/SKILL.md
  ref: 99954a30898bb5ddc8152ee10e124dc3b4106bf7
  content_hash: sha256:e84a658b65ebd997ad0a3e5553a6d03553f45737bb656e02f2109ea988774ef2
provenance:
  stars: 508
  license: Apache-2.0
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Sets up Crashlytics on Android, iOS or web, customizes crash reports, and reads crash data via MCP.
tags:
- practice
- observability
---
Loads firebase-crashlytics before a Firebase Crashlytics MCP tool is called, so it reads crash data with the right filters and report types.
