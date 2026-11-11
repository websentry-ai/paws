---
name: controlnet-pose
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+(tongyi-mai/z-image/turbo/controlnet|kling/kling-2-6/motion-control)
action:
  type: require-skill
  skill: controlnet-pose
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: controlnet-pose/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:ee223775067bac316e640f900acb766544dfd9f929a387ceed2a5e84e771439b
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Pose- or motion-conditioned generation on RunComfy via Kling Motion Control, Wan 2.2 Animate or Z-Image ControlNet.
tags:
- practice
- media-generation
---
Loads controlnet-pose before a RunComfy pose or motion-control run, so the control image and motion inputs are prepared correctly.
