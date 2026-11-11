---
name: solana-dev
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(anchor\s+(init|new|build|test|deploy|upgrade|migrate|idl|keys|verify|expand|clean)|solana\s+(program|deploy)|solana-test-validator|surfpool\s+(start|run|simulate|mcp|init|ls)|spl-token\s+(create-token|create-account|mint))($|[^\w-])
action:
  type: require-skill
  skill: solana-dev
upstream:
  repo: github.com/solana-foundation/solana-dev-skill
  path: skills/solana-dev/SKILL.md
  ref: c9da2624cc6baca5c8c2123ea50d92ef19f1c917
  content_hash: sha256:0f3eb0a58a81d12e9848dd805f1eaffa2783d14db9430aee571598e4b693fff9
provenance:
  stars: 590
  license: MIT
  last_push: '2026-11-04'
  fetched: '2026-11-12'
tags:
- practice
- blockchain
---
Loads solana-dev before a Solana build, deploy or test command or a Solana MCP call, so current kit, Anchor and testing practices are used.
