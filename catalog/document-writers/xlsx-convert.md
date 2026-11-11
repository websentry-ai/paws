---
name: xlsx-convert
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(markitdown|soffice|libreoffice|unzip|zip|python3?\s+\S*scripts/\S+)\s[^;&|\n]*\.(xlsx|xlsm|xltx)($|[\s\"'])
action:
  type: require-skill
  skill: xlsx
upstream:
  repo: github.com/anthropics/skills
  path: skills/xlsx/SKILL.md
  ref: 2c5958ecac7cee2b583967a2c4981d0f4a75c579
  content_hash: sha256:3572b018005308f765480622987ef576fcdf9ea332d1024d3fe6c58cd65566a1
provenance:
  stars: 192604
  license: NOASSERTION
  last_push: '2026-11-06'
  fetched: '2026-11-12'
summary: Opens, edits, builds and recalculates spreadsheets with formulas and formatting, checking for formula errors.
tags:
- practice
- document-writers
---
Loads xlsx before an Excel workbook is read, written, converted or recalculated, so it keeps formulas live and checks for formula errors.
