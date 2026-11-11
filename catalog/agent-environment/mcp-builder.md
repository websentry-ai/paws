---
name: mcp-builder
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npm\s+(i|install|add)|pnpm\s+(i|install|add)|yarn\s+(global\s+)?add|bun\s+(i|install|add))(\s+[^;&|\n]*)?\s+(@modelcontextprotocol/sdk|fastmcp)(@[^\s;&|)]*)?(\s|$|[;&|)])|(^|[;&|\n(])\s*(sudo\s+)?((python3?\s+-m\s+)?pip3?\s+install|uv\s+pip\s+install|uv\s+add|poetry\s+add)(\s+[^;&|\n]*)?\s+\"?(fastmcp|mcp(\[cli\])?)\"?([=<>~!]\S*)?($|[\s;&|)])
action:
  type: require-skill
  skill: mcp-builder
upstream:
  repo: github.com/anthropics/skills
  path: skills/mcp-builder/SKILL.md
  ref: 2c5958ecac7cee2b583967a2c4981d0f4a75c579
  content_hash: sha256:5a699fa929d99d7e6b1ebe10f13bf960d9ad452b70bd0733efd5d1de9524e285
provenance:
  stars: 192604
  license: NOASSERTION
  last_push: '2026-11-06'
  fetched: '2026-11-12'
summary: Builds MCP servers in Python (FastMCP) or TypeScript with well-designed tools, then writes evaluations.
tags:
- practice
- agent-environment
---
Loads mcp-builder before an MCP server SDK is installed, so the server follows MCP tool design and evaluation practice.
