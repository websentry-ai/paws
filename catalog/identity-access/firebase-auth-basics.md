---
name: firebase-auth-basics
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--yes\s+)?|bunx\s+|pnpm\s+dlx\s+)?firebase(-tools(@\S*)?)?\s+(-\S+(\s+[^-\s]\S*)?\s+)*(init\s+auth\b|deploy\b[^;&|\n]*--only[\s=]+([\w:-]*,)*auth\b)
action:
  type: require-skill
  skill: firebase-auth-basics
upstream:
  repo: github.com/firebase/agent-skills
  path: skills/firebase-auth-basics/SKILL.md
  ref: 99954a30898bb5ddc8152ee10e124dc3b4106bf7
  content_hash: sha256:3ef272cfc3d28b4dfff7d05a6eb6893d4882c97021bd8d721a12fe92ecd2a50e
provenance:
  stars: 508
  license: Apache-2.0
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Sets up Firebase Authentication providers, sign-in flows and auth-based security rules.
tags:
- practice
- identity-access
---
Loads firebase-auth-basics before Firebase Auth is initialised or deployed, so providers and rules follow the skill.
