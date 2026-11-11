---
name: video-extend
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+google-deepmind/veo-3-1/(fast/)?extend-video($|[\s;&|)\'"])
action:
  type: require-skill
  skill: video-extend
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: video-extend/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:f8b664e90b323a126711e6ae361971affb2685c67ac40beb101e5a8b0fe08d90
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Continues an existing clip with Veo 3.1 extend-video on RunComfy, keeping motion, lighting and identity consistent.
tags:
- practice
- media-generation
---
Loads video-extend before a RunComfy Veo 3.1 extend-video run, so motion, lighting and identity continuity inputs are right.
