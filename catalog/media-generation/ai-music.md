---
name: ai-music
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?@runcomfy/cli|runcomfy)\s+run\s+(elevenlabs/elevenlabs/music-generation|acestep-ai/)
action:
  type: require-skill
  skill: ai-music
upstream:
  repo: github.com/gencraft-labs/skills
  path: ai-music/SKILL.md
  ref: 5d625f59c710d7a33316b50b302b7158cdee5971
  content_hash: sha256:b244d1aac79a5c32c6352e789442db2158dccc638e8841c93c0e60a2487e9e8d
provenance:
  stars: 3
  license: MIT
  last_push: '2026-10-06'
  fetched: '2026-11-12'
summary: Routes music requests to ElevenLabs Music or ACE Step on RunComfy and gives the prompt patterns and runcomfy run call.
tags:
- practice
- media-generation
---
Loads ai-music before a RunComfy music model is run, so it uses the right model for the intent and its prompt pattern.
