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
summary: 'Careful Vercel deploys: previews freely, production with a rollback plan, and no purchase without the price and a yes.'
tags: [org, deploy]
---
Loads the vercel skill before a Vercel deploy or any purchase through the Vercel MCP server.
