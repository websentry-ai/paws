---
name: flux-kontext
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+blackforestlabs/flux-1-kontext/
action:
  type: require-skill
  skill: flux-kontext
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: flux-kontext/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:eab1ab480afb5263662cad6435e83d5a35a465aed71e26a7406ef89bae5fbb1f
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Makes precise single-reference local image edits with Flux 1 Kontext Pro on RunComfy using its prompt patterns.
tags:
- practice
- media-generation
---
Loads flux-kontext before a RunComfy Flux 1 Kontext edit run, so its single-reference prompt patterns are used.
