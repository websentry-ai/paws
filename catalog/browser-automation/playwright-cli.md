---
name: playwright-cli
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--no-install\s+)?|bunx\s+|pnpm\s+(dlx|exec)\s+)?playwright-cli\s+[a-z]
action:
  type: require-skill
  skill: playwright-cli
upstream:
  repo: github.com/microsoft/playwright-cli
  path: skills/playwright-cli/SKILL.md
  ref: 8297c9b12432ea12c1707cd64995fc96768bdb70
  content_hash: sha256:d3c50339092b670cdda1ba2bed0bdee1c34247cdf70df475e8b7d83983a39e79
provenance:
  stars: 15117
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Automates browsers and Playwright tests from the terminal with playwright-cli: open, click, fill, snapshot.'
tags:
- practice
- browser-automation
---
Loads playwright-cli before the playwright-cli command runs, so snapshot refs and command syntax are used correctly.
