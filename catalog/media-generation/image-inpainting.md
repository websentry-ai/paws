---
name: image-inpainting
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+tongyi-mai/z-image/turbo/inpainting($|[\s;&|)\'"])
action:
  type: require-skill
  skill: image-inpainting
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: image-inpainting/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:cbc7bf87bade015b022f683f7d8e1e11f7c3c35bba04d9b6a5faeb7de6945bbb
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Mask-driven inpainting on RunComfy with Z-Image Turbo Inpainting, or prose edit models when no mask exists.
tags:
- practice
- media-generation
---
Loads image-inpainting before a RunComfy Z-Image Turbo inpainting run, so mask, strength and control-scale inputs are set correctly.
