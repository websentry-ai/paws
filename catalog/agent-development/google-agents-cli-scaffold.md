---
name: google-agents-cli-scaffold
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(uvx\s+google-agents-cli|agents-cli)\s+scaffold\b
action:
  type: require-skill
  skill: google-agents-cli-scaffold
upstream:
  repo: github.com/google/agents-cli
  path: skills/google-agents-cli-scaffold/SKILL.md
  ref: 896e4719c4184841325a301755781d6b3da233ef
  content_hash: sha256:3ba305375a3a35d1d8a8c91e89103cd151b70081d0b0f167f027eade124219c2
provenance:
  stars: 6337
  license: Apache-2.0
  last_push: '2026-11-04'
  fetched: '2026-11-12'
summary: Creates, enhances or upgrades agent projects with agents-cli scaffold, choosing templates and deployment targets.
tags:
- practice
- agent-development
---
Loads google-agents-cli-scaffold before agents-cli scaffolds a project, so template and deployment target are chosen first.
