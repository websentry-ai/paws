---
name: turnstile-spin-install
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npm\s+(i|install|add)|pnpm\s+(i|install|add)|yarn\s+(global\s+)?add|bun\s+(i|install|add))(\s+((--filter|-F|-w|--workspace|--prefix|-C|--dir|--cwd|--registry)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|-d|--dev|--exact|-W|--ignore-workspace-root-check|--legacy-peer-deps|--no-save|--ignore-scripts|[^\s;&|-][^\s;&|]*))*\s+(@[\w.-]+/)?[\w.-]*turnstile[\w.-]*(@[^\s;&|)]*)?(\s|$|[;&|)])
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
