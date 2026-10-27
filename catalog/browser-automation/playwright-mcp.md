---
name: playwright-mcp
event: pre-tool
matcher:
  tool: mcp__playwright__browser_fill_form|mcp__playwright__browser_file_upload|mcp__playwright__browser_click|mcp__claude-in-chrome__form_input|mcp__claude-in-chrome__file_upload|mcp__claude-in-chrome__computer
action:
  type: require-skill
  skill: playwright
  on_miss: digest
  digest: digests/playwright.md
tags: [org, browser]
---
Loads the playwright skill before browser form fills, uploads and clicks, so credentials stay out of the agent's hands and irreversible submits get confirmed.
