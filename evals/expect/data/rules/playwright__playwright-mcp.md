---
name: "playwright-mcp"
event: "pre-tool"
matcher:
  tool: "mcp__playwright__browser_navigate|mcp__plugin_playwright_playwright__browser_navigate|mcp__claude-in-chrome__navigate"
action:
  type: require-skill
  skill: "playwright"
---
The agent opens a page with Playwright or Chrome MCP
