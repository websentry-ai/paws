---
name: scaffold-exercises
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/exercises/??-*/??.??-*/*/readme.md'
action:
  type: require-skill
  skill: scaffold-exercises
upstream:
  repo: github.com/mattpocock/skills
  path: skills/misc/scaffold-exercises/SKILL.md
  ref: 4ae2317ddb22b046e26455e4f9c9b3aaaf62212d
  content_hash: sha256:019bbcc250992c776244aa1c818aed4015364f65bb9d1dbfd3bef9bae196b065
provenance:
  stars: 321219
  license: MIT
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Scaffolds course exercise folders (problem, solution, explainer readmes) that pass the ai-hero-cli lint.
tags:
- practice
- writing
---
Loads scaffold-exercises before an exercise variant readme is written, so folder names and lint rules for the course layout are followed.
