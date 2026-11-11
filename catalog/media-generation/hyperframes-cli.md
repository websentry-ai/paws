---
name: hyperframes-cli
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?|bunx\s+|pnpm\s+dlx\s+)?hyperframes\s+(preview|render|check|lint|snapshot|history|browser|timeline|share|doctor)\b
action:
  type: require-skill
  skill: hyperframes-cli
upstream:
  repo: github.com/heygen-com/hyperframes
  path: skills/hyperframes-cli/SKILL.md
  ref: e596be16c901639ce6c18032e56924a9d8e1967d
  content_hash: sha256:d1f821fecf273df7471bc9b31b470c8f5bff7dae51afaa52e4bccda38a36d325
provenance:
  stars: 82440
  license: Apache-2.0
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: Drives the HyperFrames CLI to check, preview, render, snapshot and share HTML-to-video projects.
tags:
- practice
- media-generation
---
Loads hyperframes-cli before a HyperFrames CLI command runs, so flags, JSON output and failure recovery are right.
