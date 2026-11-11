---
name: pptx-convert
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(markitdown|pandoc|soffice|libreoffice|unzip|zip|python3?\s+\S*scripts/\S+)\s[^;&|\n]*\.(pptx|potx)($|[\s\"'])
action:
  type: require-skill
  skill: pptx
upstream:
  repo: github.com/anthropics/skills
  path: skills/pptx/SKILL.md
  ref: 2c5958ecac7cee2b583967a2c4981d0f4a75c579
  content_hash: sha256:cdec30578287eed9c8cdb12391f8a464c349ad9f9ea2543b3d9ad7d73b372c5c
provenance:
  stars: 192604
  license: NOASSERTION
  last_push: '2026-11-06'
  fetched: '2026-11-12'
summary: Reads, creates and edits PowerPoint decks with scripts, validates the XML, and renders slides to check them.
tags:
- practice
- document-writers
---
Loads pptx before a PowerPoint file is read, written or converted, so it uses the right deck tooling and validation.
