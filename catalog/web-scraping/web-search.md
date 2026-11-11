---
name: web-search
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?belt\s+app\s+(run|sample)\s+(tavily|exa)/
action:
  type: require-skill
  skill: web-search
upstream:
  repo: github.com/101-skills/superpowers
  path: tools/llm/web-search/SKILL.md
  ref: becc25649700d5457772a00e5143e28ccf9e5afa
  content_hash: sha256:96410c683758788759a62abb23c03ba2925d0901202d0048512c6a5c35e53b1c
provenance:
  stars: 23
  license: NOASSERTION
  last_push: '2026-08-31'
  fetched: '2026-11-12'
summary: Searches the web and extracts page content with Tavily and Exa apps through the belt (inference.sh) CLI.
tags:
- practice
- web-scraping
---
Loads web-search before running a Tavily or Exa app through the belt CLI, so it uses the right search and extract apps and inputs.
