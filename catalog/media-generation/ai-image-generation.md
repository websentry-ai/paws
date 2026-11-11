---
name: ai-image-generation
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?belt\s+app\s+(run\s+(\S*(image|flux|seedream|imagine-art)|(\S*/)?reve($|\s))|list\s+--category[\s=]+image($|\s))
action:
  type: require-skill
  skill: ai-image-generation
upstream:
  repo: github.com/101-skills/superpowers
  path: tools/image/ai-image-generation/SKILL.md
  ref: becc25649700d5457772a00e5143e28ccf9e5afa
  content_hash: sha256:66662cff302bdc83a4ebdebb4835302eb78b85f4630562f77c94446c7fa4a966
provenance:
  stars: 23
  license: NOASSERTION
  last_push: '2026-08-31'
  fetched: '2026-11-12'
summary: Generates and edits AI images (GPT-Image, FLUX, Gemini, Seedream and more) through the inference.sh belt CLI.
tags:
- practice
- media-generation
---
Loads ai-image-generation before running an image model through the belt CLI, so it uses the right app ids and input shapes.
