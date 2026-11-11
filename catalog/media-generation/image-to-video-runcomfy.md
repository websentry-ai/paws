---
name: image-to-video-runcomfy
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?@runcomfy/cli|runcomfy)\s+run\s+\S+/image-to-video\b
action:
  type: require-skill
  skill: image-to-video
upstream:
  repo: github.com/gencraft-labs/skills
  path: image-to-video/SKILL.md
  ref: 5d625f59c710d7a33316b50b302b7158cdee5971
  content_hash: sha256:1512174a0581b45b4f40d52eea6ed50d257dae611cd5f90671aa080a6138b17b
provenance:
  stars: 3
  license: MIT
  last_push: '2026-10-06'
  fetched: '2026-11-12'
summary: Routes still-image animation to HappyHorse I2V, Wan 2.7 or Seedance 2.0 on RunComfy with prompt patterns.
tags:
- practice
- media-generation
---
Loads image-to-video before a RunComfy image-to-video run, so the model fits the request and the prompt follows its pattern.
