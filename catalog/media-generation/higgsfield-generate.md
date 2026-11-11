---
name: higgsfield-generate
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?)?higgsfield\s+(-\S+\s+)*(generate|marketing-studio)\b
action:
  type: require-skill
  skill: higgsfield-generate
upstream:
  repo: github.com/higgsfield-ai/skills
  path: higgsfield-generate/SKILL.md
  ref: 27000402070db19d958789d05b3b52b6f4718d7c
  content_hash: sha256:20fb9cdafbea6900609e4a7f5e0091ed439ba65131dd59b915ff8202a008ca57
provenance:
  stars: 1949
  license: MIT
  last_push: '2026-11-05'
  fetched: '2026-11-12'
summary: Generates images, video, 3D and audio with Higgsfield models, Marketing Studio ads and virality analysis.
tags:
- practice
- media-generation
---
Loads higgsfield-generate before a generation or Marketing Studio command runs, so model choice and flags are right.
