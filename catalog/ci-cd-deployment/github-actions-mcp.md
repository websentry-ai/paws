---
name: github-actions-mcp
event: pre-tool
matcher:
  tool: mcp__github__run_workflow|mcp__github__rerun_workflow|mcp__github__cancel_workflow_run|mcp__github__create_release
action:
  type: require-skill
  skill: github-actions
  on_miss: digest
  digest: digests/github-actions.md
tags: [org, deploy]
---
Loads the github-actions skill before a GitHub MCP workflow dispatch, rerun, cancel or release.
