---
name: convex-quickstart
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?((npm|pnpm|yarn|bun)\s+create\s+convex|(npx\s+(-y\s+)?|bunx\s+|pnpm\s+dlx\s+)create-convex)(@\S*)?($|\s)
action:
  type: require-skill
  skill: convex-quickstart
upstream:
  repo: github.com/convex/agent-skills
  path: skills/convex-quickstart/SKILL.md
  ref: f60629d7c6c1f27eb58a56260546d2445049274d
  content_hash: sha256:3eac9a178d108ca79a80816f9d29c92e3c69c84e1d971a08c722bd07e0f91e2d
provenance:
  stars: 69
  license: Apache-2.0
  last_push: '2026-10-29'
  fetched: '2026-11-12'
summary: Scaffolds and runs a minimal Next.js + Convex template from a one-line idea with an anonymous dev deployment.
tags:
- practice
- databases
---
Loads convex-quickstart before a new Convex project is created, so it uses the one-sentence template recipe instead of hand scaffolding.
