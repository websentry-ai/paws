---
name: figma-hyperframes
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(npx\s+(-y\s+)?|bunx\s+|pnpm\s+(dlx\s+|exec\s+)?)?hyperframes\s+figma\b
action:
  type: require-skill
  skill: figma
upstream:
  repo: github.com/heygen-com/hyperframes
  path: skills/figma/SKILL.md
  ref: e596be16c901639ce6c18032e56924a9d8e1967d
  content_hash: sha256:fb444340d6c24d10ed36d3b117cce8175adaff52c17aa198b9fc9b9274c0413d
provenance:
  stars: 82440
  license: Apache-2.0
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: Imports Figma assets, tokens, components and storyboard motion into a HyperFrames video via hyperframes figma.
tags:
- practice
- media-generation
---
Loads figma before a Figma import into HyperFrames runs, so tokens, access and frame handling are right.
