---
name: cloudflare-email-service
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+(--\s+)?)?wrangler(@\S*)?\s+email($|\s)
action:
  type: require-skill
  skill: cloudflare-email-service
upstream:
  repo: github.com/cloudflare/skills
  path: skills/cloudflare-email-service/SKILL.md
  ref: cbf893b954222276bf1af57e5f23379231e0efb9
  content_hash: sha256:7c3d2a100a0cf9ff344c915ed32a8e335d3157db6943b130db104b6411f496bd
provenance:
  stars: 3337
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Implements and troubleshoots Cloudflare Email Sending and Email Routing, including domains and bindings.
tags:
- practice
- communications
---
Loads cloudflare-email-service before wrangler email is run, so it follows the onboarding prerequisites and uses the current docs.
