---
name: notion-api
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(curl|wget|http|https|xh)\s(\\\n|[^;&|\n])*api\.notion\.com/v1\b
action:
  type: require-skill
  skill: notion-api
upstream:
  repo: github.com/intellectronica/agent-skills
  path: skills/notion-api/SKILL.md
  ref: 9b0e00ad1b941165e2506545bbfddafa34cf2cb8
  content_hash: sha256:44e5829527374f83b481849c19f210f5e35e8d940a7fa0792fd6d52f5ad6492f
provenance:
  stars: 301
  license: CC0-1.0
  last_push: '2026-04-25'
  fetched: '2026-11-12'
summary: Reads and writes Notion pages, databases, blocks and comments through the REST API with curl.
tags:
- practice
- knowledge-bases
---
Loads notion-api before a Notion REST call is made, so auth headers, version and pagination follow the skill.
