---
name: relight
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+qwen/qwen-edit-2509/lora/relight($|[\s;&|)\'"])
action:
  type: require-skill
  skill: relight
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: relight/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:fe5c0d8e80c2705092fd76e348994ca4aaa49ea0cbea761aba9af82f3c0964cf
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Relights a still (direction, color temperature, mood) with Qwen Edit relight LoRA or prose edit models on RunComfy.
tags:
- practice
- media-generation
---
Loads relight before a RunComfy Qwen Edit relight run, so lighting direction and mood prompts are written correctly.
