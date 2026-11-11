---
name: anti-detect-browser
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npm\s+(i|install|add)|pnpm\s+(i|install|add)|yarn\s+(global\s+)?add|bun\s+(i|install|add))(\s+[^;&|\n]*)?\s+anti-detect-browser(@[^\s;&|)]*)?(\s|$|[;&|)])|(^|[;&|\n(])\s*(sudo\s+)?((python3?\s+-m\s+)?pip3?\s+install|uv\s+pip\s+install|uv\s+add|poetry\s+add)(\s+[^;&|\n]*)?\s+\"?antibrow\"?([=<>~!]\S*)?($|[\s;&|)])
action:
  type: require-skill
  skill: anti-detect-browser
upstream:
  repo: github.com/antibrow/anti-detect-browser-skills
  path: anti-detect-browser/SKILL.md
  ref: b800e3a54d8d4f93ae5ee23c59d13b43df5d4462
  content_hash: sha256:05a4b554bdcaf8714d211f5e1ee2621ffb1737299e7df88e572f501001b91fc1
provenance:
  stars: 21
  license: MIT
  last_push: '2026-08-28'
  fetched: '2026-11-12'
summary: Drives Chromium via Playwright with real-device fingerprints, persistent per-identity profiles and proxies.
tags:
- practice
- browser-automation
---
Loads anti-detect-browser before the anti-detect-browser or antibrow package is installed, so it applies persistent profiles and the authorized-use limits.
