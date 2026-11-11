---
name: convex-create-component
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/convex/components/**/*.{ts,tsx,js}'
action:
  type: require-skill
  skill: convex-create-component
upstream:
  repo: github.com/convex/agent-skills
  path: skills/convex-create-component/SKILL.md
  ref: f60629d7c6c1f27eb58a56260546d2445049274d
  content_hash: sha256:bc9749298386bfac9ff9ee8d23e5537dcd3bcad1fb6c48bf9587d7afdf034caa
provenance:
  stars: 69
  license: Apache-2.0
  last_push: '2026-10-29'
  fetched: '2026-11-12'
summary: Builds reusable Convex components with isolated tables and app-facing APIs.
tags:
- practice
- databases
---
Loads convex-create-component before a Convex component's code is written or edited, so it keeps component tables isolated with an app-facing wrapper API.
