---
name: better-auth-best-practices
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npm\s+(i|install|add)|pnpm\s+(i|install|add)|yarn\s+(global\s+)?add|bun\s+(i|install|add))(\s+[^;&|\n]*)?\s+(better-auth|@better-auth/[\w-]+)(@[^\s;&|)]*)?(\s|$|[;&|)])|(^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+(--\s+)?)?(@better-auth/cli(@\S*)?\s+[a-z]|auth@latest\s+(migrate|generate)($|\s))
action:
  type: require-skill
  skill: better-auth-best-practices
upstream:
  repo: github.com/better-auth/skills
  path: better-auth/best-practices/SKILL.md
  ref: 20c9e88a5c007461a703f1c213572b073196113e
  content_hash: sha256:2773c3687f2cf96d659049ebf2d8c86058a33541b09360712df0dc6b1404c403
provenance:
  stars: 229
  license: NOASSERTION
  last_push: '2026-10-19'
  fetched: '2026-11-12'
summary: Configures Better Auth server and client, database adapters, sessions, plugins and env vars.
tags:
- practice
- identity-access
---
Loads better-auth-best-practices before Better Auth is installed or its CLI is run, so the server, adapter and plugin config follow the current docs.
