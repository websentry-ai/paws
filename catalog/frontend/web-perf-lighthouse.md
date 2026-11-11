---
name: web-perf-lighthouse
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+(--\s+)?)?lighthouse\s[^;&|\n]*(https?://|localhost|file://)
action:
  type: require-skill
  skill: web-perf
upstream:
  repo: github.com/cloudflare/skills
  path: skills/web-perf/SKILL.md
  ref: cbf893b954222276bf1af57e5f23379231e0efb9
  content_hash: sha256:2716bf9bfafe4d8407b55c5213a143aea248686c4af92f8b43fc35327eaae41d
provenance:
  stars: 3337
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
tags:
- practice
- frontend
---
Loads web-perf before a performance trace or Lighthouse audit is run, so it uses the right metrics, thresholds and prioritization.
