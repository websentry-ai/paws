---
name: browser-use
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_]+=\S+\s+)*(uvx\s+)?browser-use(\s+<<|\s+--doctor|\s+[a-z])
action:
  type: require-skill
  skill: browser-use
upstream:
  repo: github.com/browser-use/browser-use
  path: skills/browser-use/SKILL.md
  ref: 7fdd63c03fd342aab4fe44507c77c3221ffd43c6
  content_hash: sha256:a0ff9f179f814a49b2f28a3985c1192dcfcc9eda6d4e7def4f7b9a7f7e20418f
provenance:
  stars: 120654
  license: MIT
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: Controls a browser over CDP with the browser-use CLI and library for automation, scraping and screenshots.
tags:
- practice
- browser-automation
---
Loads browser-use before the browser-use CLI is run, so it uses the helper functions, tab rules and daemon workflow.
