---
name: supabase-2
event: pre-tool
matcher:
  tool: mcp__*upabase*__apply_migration|mcp__*upabase*__deploy_edge_function|mcp__*upabase*__create_*|mcp__*upabase*__update_*|mcp__*upabase*__delete_*|mcp__*upabase*__merge_*
action:
  type: require-skill
  skill: supabase
upstream:
  repo: github.com/supabase/agent-skills
  path: skills/supabase/SKILL.md
  ref: c9be0e931b7930f7d02126d04774d904c381e7d7
  content_hash: sha256:2fbff32fd63b7d31585cacf0ed43e5bf8d05dd20d6c0c751d256cb967a00eec1
provenance:
  stars: 3040
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Works with any Supabase product: checks the changelog and docs first, uses CLI or MCP, and verifies each fix.'
tags:
- practice
- databases
---
Loads supabase before a Supabase CLI work command or a write through the Supabase MCP, so changelog checks, security rules and verification are followed.
