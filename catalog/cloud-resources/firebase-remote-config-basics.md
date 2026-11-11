---
name: firebase-remote-config-basics
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--yes\s+)?|bunx\s+|pnpm\s+dlx\s+)?firebase(-tools(@\S*)?)?\s+(-\S+(\s+[^-\s]\S*)?\s+)*(remoteconfig:|deploy\b[^;&|\n]*--only[\s=]+([\w:-]*,)*remoteconfig\b)
action:
  type: require-skill
  skill: firebase-remote-config-basics
upstream:
  repo: github.com/firebase/agent-skills
  path: skills/firebase-remote-config-basics/SKILL.md
  ref: 99954a30898bb5ddc8152ee10e124dc3b4106bf7
  content_hash: sha256:8037b435f69dda8a20fc108b08db55e77d45ba5e92c6116cce0276e4392e9108
provenance:
  stars: 508
  license: Apache-2.0
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Manages Remote Config templates, versions and feature flags, plus in-app defaults and SDK fetch logic.
tags:
- practice
- cloud-resources
---
Loads firebase-remote-config-basics before Remote Config commands run or the template is deployed, so it handles versions, defaults and rollback safely.
