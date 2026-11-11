---
name: better-auth-best-practices
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npm\s+(i|install|add)([ \t]+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|[ \t]+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force|[^\s;&|-][^\s;&|]*))*|pnpm\s+(i|install|add)([ \t]+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|[ \t]+[^\s;&|]+)|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts|[^\s;&|-][^\s;&|]*))*|yarn\s+(global\s+)?add([ \t]+((--cwd)(=[^\s;&|]+|[ \t]+[^\s;&|]+)|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check|[^\s;&|-][^\s;&|]*))*|bun\s+(i|install|add)([ \t]+((--cwd)(=[^\s;&|]+|[ \t]+[^\s;&|]+)|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer|[^\s;&|-][^\s;&|]*))*)[ \t]+["']?(better-auth|@better-auth/[\w-]+)(@[^\s;&|)]*)?["']?(\s|$|[;&|)])|(^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+(--\s+)?)?(@better-auth/cli(@[^\s;&|)"']*)?\s+[a-z]|auth@latest\s+(migrate|generate)($|\s))
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
