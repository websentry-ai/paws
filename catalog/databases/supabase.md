---
name: supabase
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?supabase(@\S+)?\s+(db|migration|functions|gen|link|init|branches|secrets|storage|projects|inspect|test|seed|config|orgs)($|[^\w-])
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
tags:
- practice
- databases
---
Loads supabase before a Supabase CLI work command or a write through the Supabase MCP, so changelog checks, security rules and verification are followed.
