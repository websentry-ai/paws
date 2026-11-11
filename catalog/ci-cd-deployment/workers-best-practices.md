---
name: workers-best-practices
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/wrangler.{toml,json,jsonc}'
action:
  type: require-skill
  skill: workers-best-practices
upstream:
  repo: github.com/cloudflare/skills
  path: skills/workers-best-practices/SKILL.md
  ref: cbf893b954222276bf1af57e5f23379231e0efb9
  content_hash: sha256:7f941bbc549454872a4c7c261bf75a1ef7aaa69666bc85cec91e97cf40c11cc6
provenance:
  stars: 3337
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Applies Cloudflare Workers production practices when writing, reviewing or configuring Workers.
tags:
- practice
- ci-cd-deployment
---
Loads workers-best-practices before a Wrangler config file is written or edited, so compatibility dates, observability and bindings follow production practice.
