---
name: self-hosted-funnel-launch-cli
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?|(npm|pnpm|yarn|bun)\s+create\s+)autonnel(@\S+)?(\s+[^-\s]|$)
action:
  type: require-skill
  skill: self-hosted-funnel-launch
upstream:
  repo: github.com/autonnel/autonnel-skills
  path: self-hosted-funnel-launch/SKILL.md
  ref: 9749eef7d6a2b31b7e961efc7940b4986f39583f
  content_hash: sha256:0e3e2c7d6ffe6cfaab41db76cf497fa7a3346c6cb375ed496ad0fda5be76980e
provenance:
  stars: 8
  license: Apache-2.0
  last_push: '2026-11-01'
  fetched: '2026-11-12'
summary: Deploys the self-hosted Autonnel funnel builder to Cloudflare Workers or Docker and drives it over MCP.
tags:
- practice
- ci-cd-deployment
---
Loads self-hosted-funnel-launch before Autonnel is scaffolded or one of its MCP tools is called, so it uses the Workers or Docker path and the MCP write rules.
