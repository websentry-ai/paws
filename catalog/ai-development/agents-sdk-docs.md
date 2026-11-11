---
name: agents-sdk-docs
event: pre-tool
matcher:
  tool: WebFetch
  command_regex: ^https?://developers\.cloudflare\.com/agents(/|$|\?)
action:
  type: require-skill
  skill: agents-sdk
upstream:
  repo: github.com/cloudflare/skills
  path: skills/agents-sdk/SKILL.md
  ref: cbf893b954222276bf1af57e5f23379231e0efb9
  content_hash: sha256:7522d2aa3b6b850e3778af08ba3f0d70d1dc9b8c2c5da47e7f2e3f5944e4d691
provenance:
  stars: 3337
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Builds, debugs and reviews Cloudflare Agents SDK apps using the agents package.
tags:
- practice
- ai-development
---
Loads agents-sdk before the Cloudflare agents package is installed or its docs fetched, so it uses retrieval over memory for the Agents SDK.
