---
name: firebase-ai-logic-basics-init
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--yes\s+)?|bunx\s+|pnpm\s+dlx\s+)?firebase(-tools(@\S*)?)?\s+(-\S+(\s+[^-\s]\S*)?\s+)*init\s+ailogic\b
action:
  type: require-skill
  skill: firebase-ai-logic-basics
upstream:
  repo: github.com/firebase/agent-skills
  path: skills/firebase-ai-logic-basics/SKILL.md
  ref: 99954a30898bb5ddc8152ee10e124dc3b4106bf7
  content_hash: sha256:9906e96beca49b09d9f0b82048793b29414736d71952e2d5cfbc493f044fa052
provenance:
  stars: 508
  license: Apache-2.0
  last_push: '2026-11-09'
  fetched: '2026-11-12'
tags:
- practice
- cloud-resources
---
Loads firebase-ai-logic-basics before Firebase AI Logic is initialised or added to a Flutter app, so it follows setup, multimodal and chat patterns.
