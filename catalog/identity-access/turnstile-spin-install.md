---
name: turnstile-spin-install
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npm\s+(i|install|add)(\s+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force|[^\s;&|-][^\s;&|]*))*|pnpm\s+(i|install|add)(\s+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts|[^\s;&|-][^\s;&|]*))*|yarn\s+(global\s+)?add(\s+((--cwd)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check|[^\s;&|-][^\s;&|]*))*|bun\s+(i|install|add)(\s+((--cwd)(=[^\s;&|]+|\s+[^\s;&|]+)|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer|[^\s;&|-][^\s;&|]*))*)\s+(@[\w.-]+/)?[\w.-]*turnstile[\w.-]*(@[^\s;&|)]*)?(\s|$|[;&|)])
action:
  type: require-skill
  skill: turnstile-spin
upstream:
  repo: github.com/cloudflare/skills
  path: skills/turnstile-spin/SKILL.md
  ref: cbf893b954222276bf1af57e5f23379231e0efb9
  content_hash: sha256:2ec8b83caaba227444d0247df7b119a1beebff034472e1aebb511b28659d5adb
provenance:
  stars: 3337
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Sets up Cloudflare Turnstile end to end: widget, frontend snippet and server-side siteverify, then validates.'
tags:
- practice
- identity-access
---
Loads turnstile-spin before a Turnstile widget is created or a Turnstile package is installed, so it runs the end-to-end setup with server-side siteverify.
