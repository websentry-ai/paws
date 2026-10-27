---
name: playwright-mcp
event: pre-tool
matcher:
  tool: mcp__playwright__browser_navigate|mcp__plugin_playwright_playwright__browser_navigate|mcp__claude-in-chrome__navigate
action:
  type: require-skill
  skill: playwright
  on_miss: digest
  digest: digests/playwright.md
tags: [org, browser]
---
Loads the playwright skill when the agent opens a page with Playwright or Chrome MCP (navigation only, so clicks and reads in an open page don't re-trigger it): credentials stay out of the agent's hands and irreversible submits get confirmed.
