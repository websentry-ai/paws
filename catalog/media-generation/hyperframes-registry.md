---
name: hyperframes-registry
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?|bunx\s+|pnpm\s+dlx\s+)?hyperframes\s+(add|catalog)\b
action:
  type: require-skill
  skill: hyperframes-registry
upstream:
  repo: github.com/heygen-com/hyperframes
  path: skills/hyperframes-registry/SKILL.md
  ref: e596be16c901639ce6c18032e56924a9d8e1967d
  content_hash: sha256:6ac664d7d7f2ffdf299136cffb1a9bbebe21fd2b9afa8e6f44cf0c7a2d3df3b6
provenance:
  stars: 82440
  license: Apache-2.0
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: Searches and installs ready-made HyperFrames blocks (transitions, captions, charts, effects).
tags:
- practice
- media-generation
---
Loads hyperframes-registry before a registry block is searched or installed, so ready-made blocks are used instead of hand-built ones.
