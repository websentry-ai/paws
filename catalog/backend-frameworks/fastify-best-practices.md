---
name: fastify-best-practices
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npm|pnpm|yarn|bun)\s+(i|install|add)(\s+((--filter|-F|-w|--workspace|--prefix|-C|--dir|--cwd|--registry)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|-d|--dev|--exact|-W|--ignore-workspace-root-check|--legacy-peer-deps|--no-save|--ignore-scripts|[^\s;&|-][^\s;&|]*))*\s+(fastify|@fastify/[\w.-]+)(@\S+)?([\s;&|)]|$)
action:
  type: require-skill
  skill: fastify-best-practices
upstream:
  repo: github.com/mcollina/skills
  path: skills/fastify/SKILL.md
  ref: c75d70dbfbe991d8cec42aeda0b64ff14fbb9ac7
  content_hash: sha256:1e47aea2fa16ccb263e95202224d5d50cf7c782ddffa1fe1be40b98a2bd1e815
provenance:
  stars: 1968
  license: MIT
  last_push: '2026-10-27'
  fetched: '2026-11-12'
summary: 'Builds and debugs Fastify servers: routes, plugins, JSON Schema validation, hooks, errors, TypeScript.'
tags:
- practice
- backend-frameworks
---
Loads fastify-best-practices before fastify or an @fastify plugin is installed, so routes, plugins and schemas follow Fastify conventions.
