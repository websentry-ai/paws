---
name: gsap-core
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(npm|pnpm|yarn|bun)\s+(i|install|add)(\s+((--filter|-F|-w|--workspace|--prefix|-C|--dir|--cwd|--registry)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|-d|--dev|--exact|-W|--ignore-workspace-root-check|--legacy-peer-deps|--no-save|--ignore-scripts|[^\s;&|-][^\s;&|]*))*\s+(gsap|@gsap/react)(@\S+)?(\s|$)
action:
  type: require-skill
  skill: gsap-core
upstream:
  repo: github.com/greensock/gsap-skills
  path: skills/gsap-core/SKILL.md
  ref: aed9cfd3277740755f6bfc1155c7aa645403b760
  content_hash: sha256:dfc8ba849fa8b81c8cc60964ee2a5fa8db4b2a2611801cf549a5b267b8284fa5
provenance:
  stars: 18261
  license: MIT
  last_push: '2026-07-29'
  fetched: '2026-11-12'
summary: 'Writes GSAP core animations: tweens, easing, staggers, defaults and matchMedia for responsive and reduced motion.'
tags:
- practice
- frontend
---
Loads gsap-core before GSAP is installed, so tweens, easing and reduced-motion handling use the core API correctly.
