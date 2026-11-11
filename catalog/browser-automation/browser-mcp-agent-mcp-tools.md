---
name: browser-mcp-agent-mcp-tools
event: pre-tool
matcher:
  tool: mcp__*anti-detect*|mcp__*antibrow*|mcp__*anti_detect*
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
summary: Runs the anti-detect-browser MCP server so an agent drives a fingerprinted, persistent browser through tool calls.
tags:
- practice
- browser-automation
---
Loads browser-mcp-agent before an antibrow browser MCP tool is called or the MCP server is set up, so it scopes the browser and treats page content as untrusted.
