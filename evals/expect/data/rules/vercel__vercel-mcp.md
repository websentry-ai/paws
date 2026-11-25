---
name: "vercel-mcp"
event: "pre-tool"
matcher:
  tool: "mcp__vercel__deploy_to_vercel|mcp__vercel__buy_domain|mcp__vercel__buy_credits|mcp__vercel__buy_pro|mcp__vercel__buy_addon"
action:
  type: require-skill
  skill: "vercel"
---
A Vercel deploy or any purchase through the Vercel MCP server
