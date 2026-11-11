---
name: domain-modeling
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '{**/GLOSSARY.md,**/GLOSSARY-MAP.md,**/docs/adr/**/*.md,**/adr/0*.md}'
action:
  type: require-skill
  skill: domain-modeling
upstream:
  repo: github.com/mattpocock/skills
  path: skills/engineering/domain-modeling/SKILL.md
  ref: 4ae2317ddb22b046e26455e4f9c9b3aaaf62212d
  content_hash: sha256:b78689833404148c769077355e12dbe5ad996a158a1e21e58ddf169e439d9d8c
provenance:
  stars: 321219
  license: MIT
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Sharpens the project's domain language and records it in GLOSSARY.md and ADRs as decisions crystallise.
tags:
- practice
- planning
---
Loads domain-modeling before GLOSSARY.md or an ADR is edited, so terms are challenged and decisions only recorded when they are hard to reverse.
