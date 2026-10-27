---
name: ui-ux-pro-max-ui
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/*.{tsx,jsx,vue,svelte,css,scss}'
action:
  type: require-skill
  skill: ui-ux-pro-max
upstream:
  repo: github.com/nextlevelbuilder/ui-ux-pro-max-skill
  path: .claude/skills/ui-ux-pro-max/SKILL.md
  ref: 28b3b8dee8c976df4fa3e94aed4e092ad43fcb03
  content_hash: sha256:32cc9f96965c5724bd361ec4f303baa25f68a26fd1be403e5d7a01f396d18298
provenance:
  stars: 141318
  license: MIT
  last_push: 2026-10-26
  fetched: 2026-10-27
tags: [practice, frontend]
---
Loads ui-ux-pro-max before a component or stylesheet changes, so styles, palettes, type and UX rules come from its searchable design data.
