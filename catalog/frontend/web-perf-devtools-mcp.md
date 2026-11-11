---
name: web-perf-devtools-mcp
event: pre-tool
matcher:
  tool: mcp__*devtools*__performance_*|mcp__*devtools*__lighthouse_audit
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
summary: Audits page load and interaction performance, Core Web Vitals and Lighthouse scores from traces.
tags:
- practice
- frontend
---
Loads web-perf before a performance trace or Lighthouse audit is run, so it uses the right metrics, thresholds and prioritization.
