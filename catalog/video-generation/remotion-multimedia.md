---
name: remotion-multimedia
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npm|pnpm|yarn|bun)\s+(i|install|add)(\s+((--filter|-F|-w|--workspace|--prefix|-C|--dir|--cwd|--registry)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|-d|--dev|--exact|-W|--ignore-workspace-root-check|--legacy-peer-deps|--no-save|--ignore-scripts|[^\s;&|-][^\s;&|]*))*\s+(@mediabunny/[\w.-]+|mediabunny)(@\S*)?($|[\s;&|)])
action:
  type: require-skill
  skill: remotion-multimedia
upstream:
  repo: github.com/remotion-dev/skills
  path: skills/remotion-multimedia/SKILL.md
  ref: 1af7e69a2a613437d46095924057970a6a7df40d
  content_hash: sha256:8300b4afa850128722c141a9600fd2aa4a448f186e09b7ff9dbf5257545a4573
provenance:
  stars: 6813
  license: NOASSERTION
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Uses Mediabunny to read audio/video duration and dimensions and handle media in the browser.
tags:
- practice
- video-generation
---
Loads remotion-multimedia before Mediabunny is installed, so duration and dimension reads use its documented APIs.
