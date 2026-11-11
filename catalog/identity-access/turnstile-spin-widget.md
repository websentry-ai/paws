---
name: turnstile-spin-widget
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+(--\s+)?)?(\S*/)?wrangler(@\S*)?\s+turnstile\s+widget($|\s)
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
tags:
- practice
- identity-access
---
Loads turnstile-spin before a Turnstile widget is created or a Turnstile package is installed, so it runs the end-to-end setup with server-side siteverify.
