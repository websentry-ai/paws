---
name: agent-browser-mcp
event: pre-tool
matcher:
  tool: mcp__playwright__*|mcp__claude-in-chrome__*|mcp__plugin_playwright_playwright__*
action:
  type: require-skill
  skill: agent-browser
upstream:
  repo: github.com/vercel-labs/agent-browser
  path: skills/agent-browser/SKILL.md
  ref: edb40f1878864f90e14f6c11e26fdafd3fecc8dc
  content_hash: sha256:a2eb7d7bbec1d21186e279c541f6b55eb99bd8f7f63b62675039ec05b80053ea
provenance:
  stars: 44693
  license: Apache-2.0
  last_push: 2026-10-27
  fetched: 2026-10-27
tags: [practice, browser]
---
Loads agent-browser before the first Playwright or Chrome MCP call, so the agent weighs the faster agent-browser CLI first; it is require-skill, not advise, because only require-skill installs the skill with the pack, and once it is loaded the MCP calls pass.
