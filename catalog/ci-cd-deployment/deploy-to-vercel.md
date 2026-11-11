---
name: deploy-to-vercel
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?vercel(@[^\s;&|]*)?(([ \t]|\\\n)+((--scope|-S|--cwd|-A|--local-config|-Q|--global-config|-t|--token|--team)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--[\w-]+=[^\s;&|]+|-d|--debug|--no-color|--public|-f|--force|--with-cache|--no-wait|--skip-domain|-l|--logs|\.{0,2}/[^\s;&|]*|[\w.-]+/[^\s;&|]*))*(([ \t]|\\\n)+(deploy|--prod|--prebuilt|--yes|-y)($|[^\w-])|[ \t]*($|[;&|]))
action:
  type: require-skill
  skill: deploy-to-vercel
upstream:
  repo: github.com/vercel-labs/agent-skills
  path: skills/deploy-to-vercel/SKILL.md
  ref: 1a2c79d4e6c4a2283dc2c614b6538cbe1ed5ca62
  content_hash: sha256:66b99dfa063b1c405155e9bffd99c04ca3e312e81b168d540b757bb194b7400a
provenance:
  stars: 33862
  license: NOASSERTION
  last_push: '2026-10-28'
  fetched: '2026-11-12'
summary: Deploys apps to Vercel via CLI or git push, links the project, and returns the preview or production URL.
tags:
- practice
- ci-cd-deployment
---
Loads deploy-to-vercel before a Vercel CLI deploy, so the project is linked, previews are preferred and the URL is returned.
