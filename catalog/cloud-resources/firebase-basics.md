---
name: firebase-basics
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--yes\s+)?|bunx\s+|pnpm\s+dlx\s+)?firebase(-tools(@\S*)?)?\s+(-\S+(\s+[^-\s]\S*)?\s+)*(login\b|use\b|projects:|apps:)
action:
  type: require-skill
  skill: firebase-basics
upstream:
  repo: github.com/firebase/agent-skills
  path: skills/firebase-basics/SKILL.md
  ref: 99954a30898bb5ddc8152ee10e124dc3b4106bf7
  content_hash: sha256:13a496dfb9130cbb273e44f521e78106c2e59787af92d8c06eb225839d346bbf
provenance:
  stars: 508
  license: Apache-2.0
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Installs and logs in to the Firebase CLI, creates or selects projects, and downloads app config files.
tags:
- practice
- cloud-resources
---
Loads firebase-basics before the Firebase CLI is used to log in, pick a project or fetch app config, so it follows the setup prerequisites.
