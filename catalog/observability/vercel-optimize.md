---
name: vercel-optimize
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?vercel(@\S+)?\s+(metrics|usage|contract)($|[^\w-])
action:
  type: require-skill
  skill: vercel-optimize
upstream:
  repo: github.com/vercel-labs/agent-skills
  path: skills/vercel-optimize/SKILL.md
  ref: 1a2c79d4e6c4a2283dc2c614b6538cbe1ed5ca62
  content_hash: sha256:2ae540debeedf660e9148bd892b253f02b6f15ac17c9983d406ffd299691b165
provenance:
  stars: 33862
  license: NOASSERTION
  last_push: '2026-10-28'
  fetched: '2026-11-12'
summary: Audits Vercel cost and performance from production metrics first, then gives ranked, file-backed fixes.
tags:
- practice
- observability
---
Loads vercel-optimize before Vercel metrics, usage or contract data is pulled, so the metrics-first audit procedure is followed.
