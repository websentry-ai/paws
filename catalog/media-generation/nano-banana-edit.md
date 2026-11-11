---
name: nano-banana-edit
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+google/nano-banana-2/edit($|[\s;&|)\'"])
action:
  type: require-skill
  skill: nano-banana-edit
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: nano-banana-edit/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:0c55c515f2979cc4df572e7d8522b6200d48f505806e83226e2d5845c8763867
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Edits up to 20 images with Nano Banana 2 Edit on RunComfy, keeping identity and using spatial edit language.
tags:
- practice
- media-generation
---
Loads nano-banana-edit before a RunComfy Nano Banana 2 Edit run, so identity-preserving and spatial edit prompts are used.
