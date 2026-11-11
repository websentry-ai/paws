---
name: browser-mcp-agent-setup
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+(--\s+)?)?anti-detect-browser(@\S+)?\s[^;&|\n]*--mcp\b|(^|[;&|\n(])\s*(sudo\s+)?((python3?\s+-m\s+)?pip3?\s+install|uv\s+pip\s+install|uv\s+add|poetry\s+add)(\s+[^;&|\n]*)?\s+\"antibrow\[mcp\][^\"]*\"|(^|[;&|\n(])\s*(sudo\s+)?claude\s+mcp\s+add\s[^;&|\n]*(antibrow|anti-detect-browser)
action:
  type: require-skill
  skill: browser-mcp-agent
upstream:
  repo: github.com/antibrow/anti-detect-browser-skills
  path: browser-mcp-agent/SKILL.md
  ref: b800e3a54d8d4f93ae5ee23c59d13b43df5d4462
  content_hash: sha256:46df6a34c5c7390d7db035a1646c9cd0eb579d781efeb779c0e573fc0ebb45f3
provenance:
  stars: 21
  license: MIT
  last_push: '2026-08-28'
  fetched: '2026-11-12'
tags:
- practice
- browser-automation
---
Loads browser-mcp-agent before an antibrow browser MCP tool is called or the MCP server is set up, so it scopes the browser and treats page content as untrusted.
