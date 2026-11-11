---
name: higgsfield-product-photoshoot
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?)?higgsfield\s+(-\S+\s+)*product-photoshoot\b
action:
  type: require-skill
  skill: higgsfield-product-photoshoot
upstream:
  repo: github.com/higgsfield-ai/skills
  path: higgsfield-product-photoshoot/SKILL.md
  ref: 27000402070db19d958789d05b3b52b6f4718d7c
  content_hash: sha256:cd9bf8f795be535a96852b6d5c5207ddf7238fe4ada06bb226086cac721f741a
provenance:
  stars: 1949
  license: MIT
  last_push: '2026-11-05'
  fetched: '2026-11-12'
summary: Generates brand product images (studio, lifestyle, ads, try-on) through Higgsfield's photoshoot modes.
tags:
- practice
- media-generation
---
Loads higgsfield-product-photoshoot before a photoshoot is created, so the mode and prompt rules are followed.
