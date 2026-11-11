---
name: prisma-driver-adapter-implementation
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '{**/adapter-pg/src/**/*.ts,**/adapter-mariadb/src/**/*.ts,**/adapter-mssql/src/**/*.ts,**/adapter-neon/src/**/*.ts,**/adapter-planetscale/src/**/*.ts,**/adapter-libsql/src/**/*.ts,**/adapter-better-sqlite3/src/**/*.ts,**/adapter-d1/src/**/*.ts,**/adapter-ppg/src/**/*.ts,**/adapter-mysql/src/**/*.ts,**/adapter-sqlite/src/**/*.ts}'
action:
  type: require-skill
  skill: prisma-driver-adapter-implementation
upstream:
  repo: github.com/prisma/skills
  path: prisma-driver-adapter-implementation/SKILL.md
  ref: a6f192959cf43b08aa330fe1ffca090267215812
  content_hash: sha256:6786760df68cf983505e8ea085c2ffa5253a0aa70c7b9ed72031d5f106d7d44f
provenance:
  stars: 80
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Contract reference for writing Prisma 7 SQL driver adapters: transactions, savepoints, value mapping, errors.'
tags:
- practice
- databases
---
Loads prisma-driver-adapter-implementation before editing a Prisma SQL driver adapter, so transaction, savepoint and error-mapping contracts are respected.
