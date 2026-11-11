---
name: remotion-saas
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npm\s+(i|install|add)|pnpm\s+(i|install|add)|yarn\s+(global\s+)?add|bun\s+(i|install|add))(\s+((--filter|-F|-w|--workspace|--prefix|-C|--dir|--cwd|--registry)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|-d|--dev|--exact|-W|--ignore-workspace-root-check|--legacy-peer-deps|--no-save|--ignore-scripts|[^\s;&|-][^\s;&|]*))*\s+@remotion/(player|lambda|renderer|vercel|cloudrun|web-renderer)(@\S*)?($|[\s;&|)])
action:
  type: require-skill
  skill: remotion-saas
upstream:
  repo: github.com/remotion-dev/skills
  path: skills/remotion-saas/SKILL.md
  ref: 1af7e69a2a613437d46095924057970a6a7df40d
  content_hash: sha256:a7bd14b6dc7a2fa6f2064d48abe884b4cbc934965bbce224c9b6f1f56e4f96c7
provenance:
  stars: 6813
  license: NOASSERTION
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: 'Guides building an app on Remotion: picking a template, embedding the Player, and choosing a render backend.'
tags:
- practice
- video-generation
---
Loads remotion-saas before the Remotion Player or a server render package is installed, so the template and render-backend choice is informed.
