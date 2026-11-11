---
name: migration-caveman
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '{**/migrations/**/*.sql,**/migrations/**/*.py,**/migrations/**/*.rb,**/migrations/**/*.ts,**/migrations/**/*.js,**/migrations/**/*.go,**/alembic/versions/*.py,**/db/migrate/*.rb}'
action:
  type: require-skill
  skill: migration
upstream:
  repo: github.com/juliusbrussee/caveman
  path: skills/migration/SKILL.md
  ref: 3dc497a7f858c774d6e15b25e8c6120ef3a0eda8
  content_hash: sha256:e7c6efc84ff8f85d0242e7f29f51c09fd333c294cc3123ef8ee4b2aca2a93192
provenance:
  stars: 118957
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Implements reversible schema, data, API or config migrations with rollback paths and safe sequencing.
tags:
- practice
- discipline
---
Loads migration before a migration file is written or edited, so it has a rollback path and safe sequencing.
