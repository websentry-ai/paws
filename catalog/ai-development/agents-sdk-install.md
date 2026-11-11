---
name: agents-sdk-install
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npm\s+(i|install|add)|pnpm\s+(i|install|add)|yarn\s+(global\s+)?add|bun\s+(i|install|add))(\s+((--filter|-F|-w|--workspace|--prefix|-C|--dir|--cwd)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|-d|--dev|--exact|-W|--ignore-workspace-root-check|--legacy-peer-deps|--no-save|--ignore-scripts|[^\s;&|-][^\s;&|]*))*\s+agents(@[^\s;&|)]*)?(\s|$|[;&|)])|(^|[;&|\n(])\s*(sudo\s+)?(npm|pnpm|yarn|bun)\s+create\s+cloudflare(@\S*)?\s[^;&|\n]*agents-starter
action:
  type: require-skill
  skill: agents-sdk
upstream:
  repo: github.com/cloudflare/skills
  path: skills/agents-sdk/SKILL.md
  ref: cbf893b954222276bf1af57e5f23379231e0efb9
  content_hash: sha256:7522d2aa3b6b850e3778af08ba3f0d70d1dc9b8c2c5da47e7f2e3f5944e4d691
provenance:
  stars: 3337
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
tags:
- practice
- ai-development
---
Loads agents-sdk before the Cloudflare agents package is installed or its docs fetched, so it uses retrieval over memory for the Agents SDK.
