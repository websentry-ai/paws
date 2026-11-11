---
name: browser-act
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(browser-act\s+[a-z]|uv\s+tool\s+install\s+browser-act-cli($|\s))
action:
  type: require-skill
  skill: browser-act
upstream:
  repo: github.com/browser-act/skills
  path: browser-act/SKILL.md
  ref: 593316aa0f8ccd88bdf745c92b24a50641701973
  content_hash: sha256:54938fe0c268ab553be80353110a15a66f420f86151a0e78b0369e3477d2b949
provenance:
  stars: 6342
  license: MIT
  last_push: '2026-11-04'
  fetched: '2026-11-12'
summary: 'Drives a real browser with the browser-act CLI: fetch, click, fill, screenshot, capture network, manage sessions.'
tags:
- practice
- browser-automation
---
Loads browser-act before the browser-act CLI is run or installed, so it uses the skill's browser and session workflow instead of raw commands.
