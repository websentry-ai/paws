---
name: wrangler
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+(--\s+)?)?wrangler(@\S*)?\s+[a-z]
action:
  type: require-skill
  skill: wrangler
upstream:
  repo: github.com/cloudflare/skills
  path: skills/wrangler/SKILL.md
  ref: cbf893b954222276bf1af57e5f23379231e0efb9
  content_hash: sha256:182e370d8c26e6bcb896a57a63e6cab006b253b9278cc3515f7b9db2c4d07661
provenance:
  stars: 3337
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Runs and troubleshoots Wrangler for local dev, previews, deploys and resources in projects with wrangler config.
tags:
- practice
- ci-cd-deployment
---
Loads wrangler before the Wrangler CLI is run, so it uses the project's local Wrangler and the current docs.
