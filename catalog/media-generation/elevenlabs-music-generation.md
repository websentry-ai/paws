---
name: elevenlabs-music-generation
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+elevenlabs/elevenlabs/music-generation($|[\s;&|)\'"])
action:
  type: require-skill
  skill: elevenlabs-music-generation
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: elevenlabs-music-generation/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:b8cba1ef04f7b7aca58f7a46a2098434cf6d62f262c4f99f211e5dbafeaa3fdb
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Generates songs and instrumentals with ElevenLabs Music on RunComfy, with section-structured lyrics and style prompts.
tags:
- practice
- media-generation
---
Loads elevenlabs-music-generation before a RunComfy ElevenLabs Music run, so section-structured lyrics and style prompts are used.
