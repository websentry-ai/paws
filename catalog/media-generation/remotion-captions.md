---
name: remotion-captions
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*((npm\s+(i|install|add)([ \t]+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|[ \t]+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force|[^\s;&|-][^\s;&|]*))*|pnpm\s+(i|install|add)([ \t]+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|[ \t]+[^\s;&|]+)|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts|[^\s;&|-][^\s;&|]*))*|yarn\s+(global\s+)?add([ \t]+((--cwd)(=[^\s;&|]+|[ \t]+[^\s;&|]+)|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check|[^\s;&|-][^\s;&|]*))*|bun\s+(i|install|add)([ \t]+((--cwd)(=[^\s;&|]+|[ \t]+[^\s;&|]+)|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer|[^\s;&|-][^\s;&|]*))*)|(npx|bunx|pnpm\s+exec|yarn)\s+remotion\s+(i|install|add)(\s+[^\s;&|-][^\s;&|]*)*)[ \t]+["']?@remotion/(captions|install-whisper-cpp|whisper-web|openai-whisper)(@[^\s;&|)"']*)?["']?($|[\s;&|)])
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
