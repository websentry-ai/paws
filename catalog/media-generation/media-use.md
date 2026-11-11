---
name: media-use
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?|bunx\s+|pnpm\s+dlx\s+)?hyperframes\s+(media-use|media-treatment|tts|transcribe|remove-background)\b
action:
  type: require-skill
  skill: media-use
upstream:
  repo: github.com/heygen-com/hyperframes
  path: skills/media-use/SKILL.md
  ref: e596be16c901639ce6c18032e56924a9d8e1967d
  content_hash: sha256:efcc29c82f9ee57fa213a8733af71a0587f00d914db286b9bf610836ea179760
provenance:
  stars: 82440
  license: Apache-2.0
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: Finds or generates music, voiceover, images, footage and colour treatments for a HyperFrames video and mixes the audio.
tags:
- practice
- media-generation
---
Loads media-use before a HyperFrames media command runs, so assets are sourced the supported way.
