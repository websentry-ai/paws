---
name: flux-2-klein
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+blackforestlabs/flux-2-klein/
action:
  type: require-skill
  skill: flux-2-klein
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: flux-2-klein/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:0d8aa04c6addd9b94d730b61a3dddafd04f6b9401e7963c84c3a97c280d47c62
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Generates images fast with Flux 2 Klein (9B or 4B) on RunComfy, with step-count and prompt guidance.
tags:
- practice
- media-generation
---
Loads flux-2-klein before a RunComfy Flux 2 Klein run, so the step-count and variant guidance is applied.
