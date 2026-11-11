---
name: webapp-testing
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?python3?\s+\S*with_server\.py($|\s)
action:
  type: require-skill
  skill: webapp-testing
upstream:
  repo: github.com/anthropics/skills
  path: skills/webapp-testing/SKILL.md
  ref: 2c5958ecac7cee2b583967a2c4981d0f4a75c579
  content_hash: sha256:48cd9a742a59250edec6db89cf081ef5465f365e2133f6d6c5e29c380b754b7d
provenance:
  stars: 192604
  license: NOASSERTION
  last_push: '2026-11-06'
  fetched: '2026-11-12'
summary: Tests local web apps with Python Playwright scripts, starting servers via a helper and inspecting the DOM.
tags:
- practice
- testing
---
Loads webapp-testing before the with_server.py helper is run, so it follows the helper's flags and the wait-for-networkidle pattern.
