---
name: azure-messaging
event: pre-tool
matcher:
  tool: mcp__*azure*__eventhubs*|mcp__*azure*__servicebus*
action:
  type: require-skill
  skill: azure-messaging
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-messaging/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:b3458ed24bd4f44b332fe234c6a4654d682d4ce59428ae2bc4133a3913b83c4b
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
tags:
- practice
- cloud-resources
---
Loads azure-messaging before Event Hubs or Service Bus resources are inspected, so SDK failure causes like lock expiry and AMQP errors are checked.
