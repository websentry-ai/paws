---
name: vercel-mcp
event: pre-tool
matcher:
  tool: mcp__vercel__deploy_to_vercel|mcp__vercel__buy_domain|mcp__vercel__buy_credits|mcp__vercel__buy_pro|mcp__vercel__buy_addon
action:
  type: require-skill
  skill: vercel
  on_miss: digest
  digest: digests/vercel.md
tags: [org, deploy]
---
Loads the vercel skill before a Vercel deploy or any purchase through the Vercel MCP server.
