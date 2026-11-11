---
name: playwright-stealth-verify
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npm|pnpm|yarn|bun)\s+(i|install|add)\b[^;&|\n]*\sliarjs(@\S+)?([\s;&|)]|$)
action:
  type: require-skill
  skill: playwright-stealth-verify
upstream:
  repo: github.com/liarjsdev/liarjs-skills
  path: skills/playwright-stealth-verify/SKILL.md
  ref: 4068c79209753c74f895a4a5a6d117fcde8d94d1
  content_hash: sha256:794f047636228a758e0aada415c693c59a795dbabc3e452ad3fabca1916866b0
provenance:
  stars: 5
  license: MIT
  last_push: '2026-08-06'
  fetched: '2026-11-12'
summary: Checks a Playwright or Puppeteer page's fingerprint coherence with the liarjs library, including test assertions.
tags:
- practice
- browser-automation
---
Loads playwright-stealth-verify before liarjs is added to a project, so the checkPage assertions and harness checks are known up front.
