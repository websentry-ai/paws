---
name: nx-workspace
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?((npx|bunx|pnpx)\s+(-y\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+(--\s+)?)?nx\s+(show\s+projects?\b|graph(\s|$))
action:
  type: require-skill
  skill: nx-workspace
upstream:
  repo: github.com/nrwl/nx-ai-agents-config
  path: skills/nx-workspace/SKILL.md
  ref: 13282936b5f6aa53e3d41ba589a4a2dc0d6f5786
  content_hash: sha256:f3353821bc40fef50e72ae05cce3efcf1f187b52d515239e728a2f29c04175e3
provenance:
  stars: 30
  license: MIT
  last_push: '2026-10-29'
  fetched: '2026-11-12'
summary: 'Explores an Nx workspace: projects, targets, configuration and dependency graph.'
tags:
- practice
- build-systems
---
Loads nx-workspace before nx show or nx graph is used, so workspace exploration relies on resolved config rather than project.json.
