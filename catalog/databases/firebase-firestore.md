---
name: firebase-firestore
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--yes\s+)?|bunx\s+|pnpm\s+dlx\s+)?firebase(-tools(@\S*)?)?\s+(-\S+(\s+[^-\s]\S*)?\s+)*(firestore:|deploy\b[^;&|\n]*--only[\s=]+["']?([\w:-]*,)*firestore\b|init\s+firestore\b)
action:
  type: require-skill
  skill: firebase-firestore
upstream:
  repo: github.com/firebase/agent-skills
  path: skills/firebase-firestore/SKILL.md
  ref: 99954a30898bb5ddc8152ee10e124dc3b4106bf7
  content_hash: sha256:2d69073a1488d4f1b0df2f39f46fae21382acb466c9b6139ecd0215966765129
provenance:
  stars: 508
  license: Apache-2.0
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Creates and manages Cloud Firestore databases, data models, rules, indexes and SDK integrations.
tags:
- practice
- databases
---
Loads firebase-firestore before Firestore commands run or Firestore rules and indexes are deployed, so it follows data model, index and rules practice.
