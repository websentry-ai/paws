---
name: firebase-hosting-basics
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--yes\s+)?|bunx\s+|pnpm\s+dlx\s+)?firebase(-tools(@\S*)?)?\s+(-\S+(\s+[^-\s]\S*)?\s+)*(hosting:|deploy\b[^;&|\n]*--only[\s=]+["']?([\w:-]*,)*hosting\b|emulators:start\b[^;&|\n]*--only[\s=]+["']?([\w:-]*,)*hosting\b|init\s+hosting\b)
action:
  type: require-skill
  skill: firebase-hosting-basics
upstream:
  repo: github.com/firebase/agent-skills
  path: skills/firebase-hosting-basics/SKILL.md
  ref: 99954a30898bb5ddc8152ee10e124dc3b4106bf7
  content_hash: sha256:2953792aae1bf5080490422e112208cf02a261f8c1390ad57300dcc333d00b26
provenance:
  stars: 508
  license: Apache-2.0
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: 'Deploys and configures classic Firebase Hosting: rewrites, redirects, headers, custom domains, preview channels.'
tags:
- practice
- ci-cd-deployment
---
Loads firebase-hosting-basics before classic Firebase Hosting is deployed or configured, so it uses the right rewrites, headers and preview channels.
