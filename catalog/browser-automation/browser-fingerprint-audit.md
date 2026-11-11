---
name: browser-fingerprint-audit
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?|bunx\s+|pnpm\s+(dlx|exec)\s+|yarn\s+(dlx\s+)?)?liarjs(@\S+)?(\s|$)
action:
  type: require-skill
  skill: browser-fingerprint-audit
upstream:
  repo: github.com/liarjsdev/liarjs-skills
  path: skills/browser-fingerprint-audit/SKILL.md
  ref: 4068c79209753c74f895a4a5a6d117fcde8d94d1
  content_hash: sha256:6291f135d1e2e9fe9efbc47d7ab82df90c7bc3b01f27cbbc759490ffb721c194
provenance:
  stars: 5
  license: MIT
  last_push: '2026-08-06'
  fetched: '2026-11-12'
summary: Scans a browser fingerprint with the liarjs CLI and reports where JS-layer and network-layer signals disagree.
tags:
- practice
- browser-automation
---
Loads browser-fingerprint-audit before the liarjs scan runs, so the scan flags and what it does to the machine are known.
