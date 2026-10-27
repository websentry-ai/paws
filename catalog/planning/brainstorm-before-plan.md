---
name: brainstorm-before-plan
event: pre-tool
matcher:
  tool: Write
  file_glob: '{PLAN.md,SPEC.md,PLAN-*.md,SPEC-*.md,plan.md,spec.md,plan-*.md,spec-*.md,docs/plans/**}'
action:
  type: require-skill
  skill: brainstorming
upstream:
  repo: github.com/obra/superpowers
  path: skills/brainstorming/SKILL.md
  ref: 2ce8aa15d05bd57213e6f746f953aafb7ded7a8b
  content_hash: sha256:68dc20afa48ad6fd41b034a14cbe236adacbe8fb2fdb90fdd5b6efb381e2774e
provenance:
  stars: 305962
  license: MIT
  last_push: 2026-10-23
  fetched: 2026-10-27
tags: [practice, planning]
---
Loads superpowers' brainstorming before a plan or spec file is written, so the design is explored with the user before it is written down.
