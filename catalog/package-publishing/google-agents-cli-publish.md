---
name: google-agents-cli-publish
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(agents-cli\s+publish\b|gcloud\s+(alpha\s+|beta\s+)?agent-registry\b)
action:
  type: require-skill
  skill: google-agents-cli-publish
upstream:
  repo: github.com/google/agents-cli
  path: skills/google-agents-cli-publish/SKILL.md
  ref: 896e4719c4184841325a301755781d6b3da233ef
  content_hash: sha256:821081bb378a89f3dff2965b89f1a78f3bced9845d0d1fefd4bee311c05a3f4d
provenance:
  stars: 6337
  license: Apache-2.0
  last_push: '2026-11-04'
  fetched: '2026-11-12'
summary: Registers deployed agents with Gemini and manages Agent Registry entries and MCP server registrations.
tags:
- practice
- package-publishing
---
Loads google-agents-cli-publish before an agent is registered or the Agent Registry is changed, so flags and registration mode are right.
