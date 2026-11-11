---
name: remotion-captions
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*((npm|pnpm|yarn|bun)\s+(i|install|add)|(npx|bunx|pnpm\s+exec|yarn)\s+remotion\s+(i|install|add)(\s+[^\s;&|-][^\s;&|]*)*)\s@remotion/(captions|install-whisper-cpp|whisper-web|openai-whisper)(@\S*)?($|[\s;&|)])
action:
  type: require-skill
  skill: remotion-captions
upstream:
  repo: github.com/remotion-dev/skills
  path: skills/remotion-captions/SKILL.md
  ref: 1af7e69a2a613437d46095924057970a6a7df40d
  content_hash: sha256:2d9394f042195359d3f519ba2b828940f31deb88e34acdc3c37570113336b5b2
provenance:
  stars: 6813
  license: NOASSERTION
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Transcribes audio to Remotion Caption JSON, then displays, animates or imports captions in a video.
tags:
- practice
- media-generation
---
Loads remotion-captions before a Remotion captions or whisper package is installed, so captions use the Caption JSON format.
