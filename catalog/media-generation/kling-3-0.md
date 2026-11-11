---
name: kling-3-0
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+kling/kling-3\.0/
action:
  type: require-skill
  skill: kling-3-0
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: kling-3-0/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:111704adf0de8d97098e311917b26ab9748f22c99e091d23e9ca4758d62a1f18
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Generates text- or image-to-video with Kling 3.0 (Standard, Pro, 4K) on RunComfy with native audio.
tags:
- practice
- media-generation
---
Loads kling-3-0 before a RunComfy Kling 3.0 run, so the tier, mode and native audio options are chosen correctly.
