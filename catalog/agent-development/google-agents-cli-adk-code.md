---
name: google-agents-cli-adk-code
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(((python3?\s+-m\s+)?pip3?|uv\s+pip)\s+install|uv\s+add|poetry\s+add)([ \t]+((-t|--target|--prefix|-r|--requirement|-c|--constraint|-i|--index-url|--extra-index-url|--root|--src|--group|--extra|--optional)(=[^\s;&|]+|[ \t]+[^\s;&|]+)|-U|--upgrade|--user|-q|--quiet|--no-deps|--pre|--no-cache-dir|--force-reinstall|-I|--ignore-installed|--dev|-D|[^\s;&|-][^\s;&|]*))*[ \t]+["']?[\"']?google-adk(\[[^\]]*\])?([\"']|[=<>~!@]\S*)?["']?($|[\s;&|)])
action:
  type: require-skill
  skill: google-agents-cli-adk-code
upstream:
  repo: github.com/google/agents-cli
  path: skills/google-agents-cli-adk-code/SKILL.md
  ref: 896e4719c4184841325a301755781d6b3da233ef
  content_hash: sha256:22d285723230a4c81e9cd44387725eafbe45ffcc9df83429ba155a7afad95c6d
provenance:
  stars: 6337
  license: Apache-2.0
  last_push: '2026-11-04'
  fetched: '2026-11-12'
summary: 'Quick reference and recipes for writing ADK agent code: agent types, tools, callbacks, state and workflows.'
tags:
- practice
- agent-development
---
Loads google-agents-cli-adk-code before the ADK package is installed, so agent code follows ADK patterns from the start.
