---
name: firebase-app-hosting-basics-yaml
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/apphosting*.yaml'
action:
  type: require-skill
  skill: firebase-app-hosting-basics
upstream:
  repo: github.com/firebase/agent-skills
  path: skills/firebase-app-hosting-basics/SKILL.md
  ref: 99954a30898bb5ddc8152ee10e124dc3b4106bf7
  content_hash: sha256:1df810f9390362025772649f5967512e1f27735139ce0a38bc6834d4a440809a
provenance:
  stars: 508
  license: Apache-2.0
  last_push: '2026-11-09'
  fetched: '2026-11-12'
tags:
- practice
- ci-cd-deployment
---
Loads firebase-app-hosting-basics before Firebase App Hosting is deployed or apphosting.yaml is edited, so it follows backend, secret and rollout rules.
