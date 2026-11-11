---
name: lark-approval
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+approval\b
action:
  type: require-skill
  skill: lark-approval
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-approval/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:343b83219d633fffe98561c2e46151a734ba0369a582d7ebd7b9dc143ae25908
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Queries and processes Lark/Feishu approval tasks and instances, and starts new approvals via lark-cli.
tags:
- practice
- workplace-suite
---
Loads lark-approval before lark-cli handles approval tasks or starts an approval, so actions go to the right instance.
