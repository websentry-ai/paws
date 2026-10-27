---
name: writing-plans-before-plan
event: pre-tool
matcher:
  tool: Write
  file_glob: '{PLAN.md,SPEC.md,PLAN-*.md,SPEC-*.md,plan.md,spec.md,plan-*.md,spec-*.md,docs/plans/**}'
action:
  type: require-skill
  skill: writing-plans
upstream:
  repo: github.com/obra/superpowers
  path: skills/writing-plans/SKILL.md
  ref: 2ce8aa15d05bd57213e6f746f953aafb7ded7a8b
  content_hash: sha256:ef082ddb5354d736059397133b3682e8a0c85c57906affe47c9da354cbb7c91e
provenance:
  stars: 305962
  license: MIT
  last_push: 2026-10-23
  fetched: 2026-10-27
tags: [practice, planning]
---
Loads superpowers' writing-plans, which brainstorming hands off to, so the plan file follows its structure.
