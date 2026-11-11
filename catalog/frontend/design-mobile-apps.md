---
name: design-mobile-apps
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(curl|wget|http|https|xh)\s[^;&|\n]*https?://([\w-]+\.)*sleek\.design/api/
action:
  type: require-skill
  skill: design-mobile-apps
upstream:
  repo: github.com/designed-by-ai/skills
  path: skills/design-mobile-apps/SKILL.md
  ref: 03c20aec6dcedb5fabfb03dc37982eb52ef19ed6
  content_hash: sha256:eec6dae3f26f8f154fe190a53387f0b2eab39aa120932d5c33af7f3284f04b10
provenance:
  stars: 30
  license: MIT
  last_push: '2026-10-06'
  fetched: '2026-11-12'
summary: Designs mobile app screens through the Sleek REST API and implements them in React Native or HTML.
tags:
- practice
- frontend
---
Loads design-mobile-apps before the Sleek REST API is called from the shell, so it follows the envelope, id and polling rules.
