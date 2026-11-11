---
name: durable-objects
event: pre-tool
matcher:
  tool: WebFetch
  command_regex: ^https?://developers\.cloudflare\.com/(durable-objects|workers/runtime-apis/durable-objects)(/|$|\?)
action:
  type: require-skill
  skill: durable-objects
upstream:
  repo: github.com/cloudflare/skills
  path: skills/durable-objects/SKILL.md
  ref: cbf893b954222276bf1af57e5f23379231e0efb9
  content_hash: sha256:659669f02a6e1b37020b799be0f71e3fecc7b5111f3f406d9f844a0a21994113
provenance:
  stars: 3337
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Builds, debugs and reviews Cloudflare Durable Objects code for persistent state and coordination.
tags:
- practice
- cloud-resources
---
Loads durable-objects before the Cloudflare Durable Objects docs are fetched, so it applies the rules for storage, alarms, RPC and testing.
