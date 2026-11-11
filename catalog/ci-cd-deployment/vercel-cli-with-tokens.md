---
name: vercel-cli-with-tokens
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?vercel(@\S+)?\s[^;&|\n]*--token($|[^\w-])|(^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*VERCEL_TOKEN=\S*\s+([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?vercel(@\S+)?($|[^\w-])
action:
  type: require-skill
  skill: vercel-cli-with-tokens
upstream:
  repo: github.com/vercel-labs/agent-skills
  path: skills/vercel-cli-with-tokens/SKILL.md
  ref: 1a2c79d4e6c4a2283dc2c614b6538cbe1ed5ca62
  content_hash: sha256:a294080c5464284c5ed06bde9e5bf0fb16981958600f4ae829eaab5d7f17d29a
provenance:
  stars: 33862
  license: NOASSERTION
  last_push: '2026-10-28'
  fetched: '2026-11-12'
summary: Deploys and manages Vercel projects with access tokens instead of interactive login, including env vars.
tags:
- practice
- ci-cd-deployment
---
Loads vercel-cli-with-tokens before a Vercel CLI call that passes a token, so token lookup and non-interactive flags are handled safely.
