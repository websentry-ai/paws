---
name: docx-files
event: pre-tool
matcher:
  tool: Read|Write|Edit
  file_glob: '**/*.{docx,dotx}'
action:
  type: require-skill
  skill: docx
upstream:
  repo: github.com/anthropics/skills
  path: skills/docx/SKILL.md
  ref: 2c5958ecac7cee2b583967a2c4981d0f4a75c579
  content_hash: sha256:566b0933d95f5a2f981050e781162e19cebacfdff5608ab23dbe556aaf7a59e5
provenance:
  stars: 192604
  license: NOASSERTION
  last_push: '2026-11-06'
  fetched: '2026-11-12'
tags:
- practice
- document-writers
---
Loads docx before a Word file is read, written or converted, so it uses the right docx tooling, tracked-change handling and validation.
