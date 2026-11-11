---
name: remotion-to-hyperframes
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?|bunx\s+|pnpm\s+dlx\s+)?hyperframes\s+diff\b
action:
  type: require-skill
  skill: remotion-to-hyperframes
upstream:
  repo: github.com/heygen-com/hyperframes
  path: skills/remotion-to-hyperframes/SKILL.md
  ref: e596be16c901639ce6c18032e56924a9d8e1967d
  content_hash: sha256:2f3e17b0a2770abba9e830512d2c0ca1aba5db2469c95d51ab1c87d5a29e849f
provenance:
  stars: 82440
  license: Apache-2.0
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: Ports Remotion, Lottie, Three.js or other animation code into HyperFrames and checks it frame by frame.
tags:
- practice
- media-generation
---
Loads remotion-to-hyperframes before a ported video is compared with the original, so the verification steps are followed.
