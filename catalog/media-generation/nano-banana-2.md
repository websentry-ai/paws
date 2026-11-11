---
name: nano-banana-2
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+google/nano-banana-2/text-to-image($|[\s;&|)\'"])
action:
  type: require-skill
  skill: nano-banana-2
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: nano-banana-2/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:94218dc373980e0d80578f09d7155371d279387cd2d3a61a0321445d2a802160
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Prompts Google Nano Banana 2 text-to-image on RunComfy with its documented patterns and resolution and safety settings.
tags:
- practice
- media-generation
---
Loads nano-banana-2 before a RunComfy Nano Banana 2 text-to-image run, so the prompt patterns and safety and resolution settings are applied.
