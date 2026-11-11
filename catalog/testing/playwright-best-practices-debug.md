---
name: playwright-best-practices-debug
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+(--\s+)?)?playwright\s+test\b[^;&|\n]*\s--(debug|ui|trace|repeat-each|last-failed|update-snapshots|grep|retries)\b
action:
  type: require-skill
  skill: playwright-best-practices
upstream:
  repo: github.com/currents-dev/playwright-best-practices-skill
  path: playwright-best-practices/SKILL.md
  ref: 0b75b5835a87de1fdd202ba965f3bc5a27738fdf
  content_hash: sha256:111e9d3e9d63dea34814120fb6583c0b28ee45371662801d8e8d8bdf4b05dcb0
provenance:
  stars: 422
  license: MIT
  last_push: '2026-11-08'
  fetched: '2026-11-12'
tags:
- practice
- testing
---
Loads playwright-best-practices before Playwright tests are debugged or the Playwright config is edited, so it applies locator, fixture and flakiness practice.
