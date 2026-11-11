---
name: playwright-best-practices-config
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/playwright.config.{ts,js,mjs,cjs}'
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
summary: 'Writes and debugs Playwright tests: locators, fixtures, flakiness, auth, mocking, CI, visual and a11y testing.'
tags:
- practice
- testing
---
Loads playwright-best-practices before Playwright tests are debugged or the Playwright config is edited, so it applies locator, fixture and flakiness practice.
