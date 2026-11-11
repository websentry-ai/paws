---
name: slideshow-hyperframes
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(npx\s+(-y\s+)?|bunx\s+|pnpm\s+(dlx\s+|exec\s+)?)?hyperframes\s+present\b
action:
  type: require-skill
  skill: slideshow
upstream:
  repo: github.com/heygen-com/hyperframes
  path: skills/slideshow/SKILL.md
  ref: e596be16c901639ce6c18032e56924a9d8e1967d
  content_hash: sha256:9e0fe437b47e9edf4f1c6c9aa48b5c1d7c3a1b896d9a83a11934c7aa8e76d345
provenance:
  stars: 82440
  license: Apache-2.0
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: Builds a live HyperFrames slideshow with fragments, hotspots and presenter notes, served by hyperframes present.
tags:
- practice
- media-generation
---
Loads slideshow before a HyperFrames deck is served, so the slideshow island is valid before it runs.
