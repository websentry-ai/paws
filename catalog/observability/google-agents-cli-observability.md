---
name: google-agents-cli-observability
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(uvx\s+google-agents-cli|agents-cli)\s+infra\s+single-project\b
action:
  type: require-skill
  skill: google-agents-cli-observability
upstream:
  repo: github.com/google/agents-cli
  path: skills/google-agents-cli-observability/SKILL.md
  ref: 896e4719c4184841325a301755781d6b3da233ef
  content_hash: sha256:dd70df77604ba1ff87e9397b98f2851cbb44e9a5df487c7dd1db34ec91c25c0b
provenance:
  stars: 6337
  license: Apache-2.0
  last_push: '2026-11-04'
  fetched: '2026-11-12'
summary: Sets up tracing, prompt-response logging and BigQuery analytics for deployed agents, plus third-party integrations.
tags:
- practice
- observability
---
Loads google-agents-cli-observability before telemetry infrastructure is provisioned, so logging and analytics are set up in the right order.
