---
name: seedance-v2
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+bytedance/seedance-v2/
action:
  type: require-skill
  skill: seedance-v2
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: seedance-v2/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:2eafeef4618512c9048ffae2ead8dd6922ac370e15fee6326d6af515cba529fb
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Generates cinematic short video with Seedance 2.0 Pro on RunComfy from image, video and audio references with lip-sync.
tags:
- practice
- media-generation
---
Loads seedance-v2 before a RunComfy Seedance 2.0 run, so the multi-reference and audio inputs are set up correctly.
