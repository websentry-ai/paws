---
name: higgsfield-soul-id
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?)?higgsfield\s+(-\S+\s+)*soul-id\b
action:
  type: require-skill
  skill: higgsfield-soul-id
upstream:
  repo: github.com/higgsfield-ai/skills
  path: higgsfield-soul-id/SKILL.md
  ref: 27000402070db19d958789d05b3b52b6f4718d7c
  content_hash: sha256:be1abb8875552d0e8a2aa2fb4ac9fd6a3edf55aab55ab4ceb65b611ef037ddf4
provenance:
  stars: 1949
  license: MIT
  last_push: '2026-11-05'
  fetched: '2026-11-12'
summary: Trains a Higgsfield Soul Character from face photos for identity-consistent image and video generation.
tags:
- practice
- media-generation
---
Loads higgsfield-soul-id before a Soul Character is trained, so photos and model variant are chosen correctly.
