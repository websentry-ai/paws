---
name: lark-markdown
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?lark-cli\s+markdown\b
action:
  type: require-skill
  skill: lark-markdown
upstream:
  repo: github.com/larksuite/cli
  path: skills/lark-markdown/SKILL.md
  ref: ac22a60a23d66c37d43a7ec6d9e99b99a4320356
  content_hash: sha256:5ff48e5f8525171b05ce2d62ab96e561c39221261fa55a2e959ede2be39f0b0e
provenance:
  stars: 18049
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Views, creates, uploads, edits and diffs native Markdown files stored in Lark/Feishu via lark-cli.
tags:
- practice
- document-writers
---
Loads lark-markdown before lark-cli works on native Markdown files, so create, patch and diff use the right commands.
