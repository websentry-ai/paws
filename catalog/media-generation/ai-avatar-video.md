---
name: ai-avatar-video
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?belt\s+app\s+run\s+\S*(avatar|omnihuman|lipsync|/fabric-\d)
action:
  type: require-skill
  skill: ai-avatar-video
upstream:
  repo: github.com/101-skills/superpowers
  path: tools/video/ai-avatar-video/SKILL.md
  ref: becc25649700d5457772a00e5143e28ccf9e5afa
  content_hash: sha256:0769c92ae30ffedd75f944665763a2ceb9735716f3a5432e84bd23841b9d9767
provenance:
  stars: 23
  license: NOASSERTION
  last_push: '2026-08-31'
  fetched: '2026-11-12'
summary: Creates talking-head and lipsync avatar videos (P-Video-Avatar, OmniHuman, Fabric) through the belt CLI.
tags:
- practice
- media-generation
---
Loads ai-avatar-video before running an avatar or lipsync model through the belt CLI, so it picks the right avatar model and input fields.
