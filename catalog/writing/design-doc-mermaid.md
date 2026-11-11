---
name: design-doc-mermaid
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/*.{mmd,mermaid}'
action:
  type: require-skill
  skill: design-doc-mermaid
upstream:
  repo: github.com/spillwavesolutions/design-doc-mermaid
  path: SKILL.md
  ref: c36d83503d2ba9c1e51638dc2d8a34758a377dea
  content_hash: sha256:317ac5b6fc759e726b41fd11e9925514b4e122be9d50060902b5664f8e1f1744
provenance:
  stars: 184
  license: NOASSERTION
  last_push: '2026-08-24'
  fetched: '2026-11-12'
summary: Creates Mermaid diagrams (flowchart, sequence, ER, C4) from text or source code for docs and wikis.
tags:
- practice
- writing
---
Loads design-doc-mermaid before a Mermaid diagram file is written, so diagram type choice and validation steps are followed.
