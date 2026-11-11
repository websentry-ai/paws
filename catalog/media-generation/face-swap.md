---
name: face-swap
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+community/wan-2-2-animate(/|($|[\s;&|)\'"]))
action:
  type: require-skill
  skill: face-swap
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: face-swap/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:ffeaa8523afd9058803d38793051de993b9c59097a7b6f3f8da1149c6db315fc
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Routes face and character swaps in stills or video to Wan 2.2 Animate, GPT Image 2 Edit, Flux Kontext or Kling.
tags:
- practice
- media-generation
---
Loads face-swap before a RunComfy Wan 2.2 Animate run, so the right swap route and inputs are chosen.
