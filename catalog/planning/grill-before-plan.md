---
name: grill-before-plan
event: pre-tool
matcher:
  tool: Write
  file_glob: '{PLAN.md,SPEC.md,PLAN-*.md,SPEC-*.md,plan.md,spec.md,plan-*.md,spec-*.md,docs/plans/**}'
action:
  type: require-skill
  skill: grilling
upstream:
  repo: github.com/mattpocock/skills
  path: skills/productivity/grilling/SKILL.md
  ref: 59db609eb1e18aaf59b63284d0e52255aff0c63c
  content_hash: sha256:74de5501c2b0cec8410bf4ebc9eaddc84f6f7706e04d2c70a814872b1c44e410
provenance:
  stars: 304568
  license: MIT
  last_push: 2026-10-27
  fetched: 2026-10-27
tags: [practice, planning]
---
Loads Matt Pocock's grilling (the skill his user-only /grill-me hands off to) before a plan or spec file is written, so every open decision is put to the user first.
