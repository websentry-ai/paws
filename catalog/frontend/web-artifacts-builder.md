---
name: web-artifacts-builder
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?((ba|z)?sh\s+)?(\S*/)?(init|bundle)-artifact\.sh($|\s)
action:
  type: require-skill
  skill: web-artifacts-builder
upstream:
  repo: github.com/anthropics/skills
  path: skills/web-artifacts-builder/SKILL.md
  ref: 2c5958ecac7cee2b583967a2c4981d0f4a75c579
  content_hash: sha256:7a081b86fd5068d3ca2c4d9c059956634feff65ccbf1c39c3d34a6e5d0971139
provenance:
  stars: 192604
  license: NOASSERTION
  last_push: '2026-11-06'
  fetched: '2026-11-12'
summary: Scaffolds a React/Tailwind/shadcn project and bundles it into one HTML file for a claude.ai artifact.
tags:
- practice
- frontend
---
Loads web-artifacts-builder before the artifact init or bundle script is run, so the project is scaffolded and bundled the way the skill expects.
