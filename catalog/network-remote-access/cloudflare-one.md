---
name: cloudflare-one
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(cloudflared\s+(tunnel\s+(create|route|run|login|delete|token|ingress|info|list|cleanup)|access|service\s+install)|warp-cli\s+(registration|connect|disconnect|status|settings|tunnel|dns|debug|mdm))($|\s)
action:
  type: require-skill
  skill: cloudflare-one
upstream:
  repo: github.com/cloudflare/skills
  path: skills/cloudflare-one/SKILL.md
  ref: cbf893b954222276bf1af57e5f23379231e0efb9
  content_hash: sha256:198e43c174b287819638e8d011494055d7d77c69fbf8b275b3a99c641c7bf949
provenance:
  stars: 3337
  license: Apache-2.0
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Designs, configures and troubleshoots Cloudflare One Zero Trust: Access, Gateway, WARP, Tunnel.'
tags:
- practice
- network-remote-access
---
Loads cloudflare-one before a Zero Trust tunnel, Access or WARP client command is run, so it retrieves current Cloudflare One docs and stages risky changes.
