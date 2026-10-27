---
name: linear-mcp
event: pre-tool
matcher:
  tool: mcp__linear__save_issue|mcp__linear__save_project|mcp__linear__save_comment|mcp__linear__save_initiative|mcp__linear__delete_comment
action:
  type: require-skill
  skill: linear
  on_miss: digest
  digest: digests/linear.md
tags: [org, issues]
---
Loads the linear skill before Linear issues, projects, initiatives or comments are written or deleted.
