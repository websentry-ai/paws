---
name: happyhorse-1-0
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+happyhorse/happyhorse-1-0/
action:
  type: require-skill
  skill: happyhorse-1-0
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: happyhorse-1-0/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:50c0960d5cb89dadd35572053a70702e15b0da143984e8d2094feca2e40e80ba
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Generates 1080p video with synced audio using HappyHorse 1.0 on RunComfy, with its multi-shot prompt patterns.
tags:
- practice
- media-generation
---
Loads happyhorse-1-0 before a RunComfy HappyHorse 1.0 run, so the multi-shot prompt patterns are used.
