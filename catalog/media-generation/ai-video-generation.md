---
name: ai-video-generation
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?belt\s+app\s+(run\s+\S*(veo-|wan-|seedance|happyhorse|video)|list\s+--category[\s=]+video($|\s))
action:
  type: require-skill
  skill: ai-video-generation
upstream:
  repo: github.com/101-skills/superpowers
  path: tools/video/ai-video-generation/SKILL.md
  ref: becc25649700d5457772a00e5143e28ccf9e5afa
  content_hash: sha256:8ff792abc9b85aa6b7034bf720ab49e66e78fd65497a81036c078191688c6daf
provenance:
  stars: 23
  license: NOASSERTION
  last_push: '2026-08-31'
  fetched: '2026-11-12'
summary: Generates and edits AI video (Veo, Seedance, Wan, HappyHorse and more) through the inference.sh belt CLI.
tags:
- practice
- media-generation
---
Loads ai-video-generation before running a video model through the belt CLI, so it uses the right app ids and input shapes.
