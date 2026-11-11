---
name: supabase-postgres-best-practices
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '{**/migrations/**/*.sql,**/supabase/**/*.sql}'
action:
  type: require-skill
  skill: supabase-postgres-best-practices
upstream:
  repo: github.com/supabase/agent-skills
  path: skills/supabase-postgres-best-practices/SKILL.md
  ref: c9be0e931b7930f7d02126d04774d904c381e7d7
  content_hash: sha256:3eb972c2d7c818e544e35cc33ea8d7147d4d78ae08142ef40037b0fbac7a2505
provenance:
  stars: 3040
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Postgres rules for schema, migrations, RLS, indexes, functions and query tuning, maintained by Supabase.
tags:
- practice
- databases
---
Loads supabase-postgres-best-practices before a SQL migration or Supabase schema file is edited, so schema, RLS and index rules are applied.
