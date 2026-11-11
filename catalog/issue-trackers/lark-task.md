---
name: lark-task
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+task\b
action:
  type: require-skill
  skill: lark-task
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-task/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:b79f5426a8ea438dcb057b02f70a30ecd41a5fa21fb9d2a6968a779e882771da
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Creates and updates Lark/Feishu tasks, subtasks, task lists, assignees and task agents through lark-cli.
tags:
- practice
- issue-trackers
---
Loads lark-task before lark-cli creates or updates tasks, so task, list and assignee fields are right.
