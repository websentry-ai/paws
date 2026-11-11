---
name: convex
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/convex/**/*.{ts,tsx,js,jsx}'
action:
  type: require-skill
  skill: convex
upstream:
  repo: github.com/convex/agent-skills
  path: skills/convex/SKILL.md
  ref: f60629d7c6c1f27eb58a56260546d2445049274d
  content_hash: sha256:22939725331c04868bc7bb56e49623cf9ee5b2c59b805ee3a5633be8ee9a369a
provenance:
  stars: 69
  license: Apache-2.0
  last_push: '2026-10-29'
  fetched: '2026-11-12'
summary: Routes Convex backend work (schema, functions, auth, crons, components) to the right convex-* guidance.
tags:
- practice
- databases
---
Loads convex before Convex backend code is written or edited, so it is routed to the right Convex guidance.
