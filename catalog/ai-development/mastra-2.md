---
name: mastra-2
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/src/mastra/**/*.{ts,tsx,js,mjs,json}'
action:
  type: require-skill
  skill: mastra
upstream:
  repo: github.com/mastra-ai/skills
  path: skills/mastra/SKILL.md
  ref: 6acbd14c60aedc32b4851db3448f5cc5a4a9ae3b
  content_hash: sha256:d05375a6c989d2053456143288c78e50a3a4f653f42e5bec417e708ac002fe02
provenance:
  stars: 103
  license: NOASSERTION
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Builds Mastra agents, workflows, tools and memory against current docs, and runs mastra api and deploy tasks.
tags:
- practice
- ai-development
---
Loads mastra before the Mastra CLI runs or Mastra code is written, so APIs are checked against current docs instead of memory.
