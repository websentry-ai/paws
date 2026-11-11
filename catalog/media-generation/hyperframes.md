---
name: hyperframes
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?|bunx\s+|pnpm\s+dlx\s+)?hyperframes\s+(init|capture)\b
action:
  type: require-skill
  skill: hyperframes
upstream:
  repo: github.com/heygen-com/hyperframes
  path: skills/hyperframes/SKILL.md
  ref: e596be16c901639ce6c18032e56924a9d8e1967d
  content_hash: sha256:efecc3d588e38d0ea2323f0860e431b6b1795dc2f9994ad795a0b0f86e397954
provenance:
  stars: 82440
  license: Apache-2.0
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: Holds the HyperFrames render contract and work loop for making, editing, animating or rendering a video from HTML.
tags:
- practice
- media-generation
---
Loads hyperframes before a HyperFrames project is created or captured, so the render contract is known from the first file.
