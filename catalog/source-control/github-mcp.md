---
name: github-mcp
event: pre-tool
matcher:
  tool: mcp__github__create_or_update_file|mcp__github__push_files|mcp__github__create_branch|mcp__github__create_pull_request|mcp__github__merge_pull_request|mcp__github__delete_file|mcp__github__create_repository
action:
  type: require-skill
  skill: github
  on_miss: digest
  digest: digests/github.md
tags: [org, git]
---
Loads the github skill before GitHub MCP writes: files, pushes, branches, pull requests, merges and new repositories.
