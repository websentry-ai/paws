---
name: runcomfy-cli
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*((npm|pnpm|yarn|bun)\s+(i|install|add)\s+(-g\s+|--global\s+)?@runcomfy/cli(@\S+)?($|[\s;&|)])|(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(login|logout|whoami|status)($|[^\w-])|(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+[^;&|\n]*--(no-wait|input-file)($|[^\w-]))
action:
  type: require-skill
  skill: runcomfy-cli
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: runcomfy-cli/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:7f472efd73f4172515f3301d6e06427784da690f58605a7268d8573a0b37a91b
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: 'Installs, signs in and drives the runcomfy CLI: model schemas, run, poll, no-wait, JSON output and errors.'
tags:
- practice
- media-generation
---
Loads runcomfy-cli before installing, signing in to or scripting the runcomfy CLI, so install, auth and no-wait polling are done right.
