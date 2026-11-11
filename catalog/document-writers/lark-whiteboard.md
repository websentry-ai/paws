---
name: lark-whiteboard
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(lark-cli\s+whiteboard\b|(npx\s+(-y\s+)?|bunx\s+|pnpm\s+dlx\s+)?@larksuite/whiteboard-cli\b)
action:
  type: require-skill
  skill: lark-whiteboard
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-whiteboard/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:d603481f74516ffc59f1641eb855591e0429da002aa9431277e4dbda97da6cb4
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Exports and edits whiteboards inside Lark/Feishu docs through lark-cli and the whiteboard-cli package.
tags:
- practice
- document-writers
---
Loads lark-whiteboard before whiteboards are exported or edited, so the right CLI and format are used.
