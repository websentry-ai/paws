---
name: fingerprint-ci-gate
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?|bunx\s+|pnpm\s+(dlx|exec)\s+|yarn\s+(dlx\s+)?)?liarjs(@\S+)?\s[^;&|\n]*--min-score([=\s]|$)
action:
  type: require-skill
  skill: fingerprint-ci-gate
upstream:
  repo: github.com/liarjsdev/liarjs-skills
  path: skills/fingerprint-ci-gate/SKILL.md
  ref: 4068c79209753c74f895a4a5a6d117fcde8d94d1
  content_hash: sha256:d2b6ebb4d8180541a971986448b2ad75df334298a320908e144a69be8bc6f0ef
provenance:
  stars: 5
  license: MIT
  last_push: '2026-08-06'
  fetched: '2026-11-12'
summary: Gates CI on browser fingerprint regressions with liarjs baselines, diffs and a minimum score.
tags:
- practice
- ci-cd-deployment
---
Loads fingerprint-ci-gate before liarjs is run with --min-score, so baselines, diffs and exit codes follow the gating recipe.
