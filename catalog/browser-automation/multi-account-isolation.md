---
name: multi-account-isolation
event: pre-tool
matcher:
  tool: mcp__*navigate*
  command_regex: https?://([\w-]+\.)*(browserleaks\.com|whoer\.net|pixelscan\.net|abrahamjuliot\.github\.io/creepjs|liarjs\.dev)(/|\"|\?|:)
action:
  type: require-skill
  skill: multi-account-isolation
upstream:
  repo: github.com/antibrow/anti-detect-browser-skills
  path: multi-account-isolation/SKILL.md
  ref: b800e3a54d8d4f93ae5ee23c59d13b43df5d4462
  content_hash: sha256:b97d1df30750b9f7d846f65fcb6a6e22b6f0a43be91a3bd419196885ca046c0e
provenance:
  stars: 21
  license: MIT
  last_push: '2026-08-28'
  fetched: '2026-11-12'
summary: 'Checks browser profiles are truly isolated: timezone vs IP, WebRTC leaks, canvas hashes, shared personas.'
tags:
- practice
- browser-automation
---
Loads multi-account-isolation before a browser is pointed at a fingerprint test site such as CreepJS or browserleaks, so the isolation checks (timezone, WebRTC, canvas hash) are run properly.
