---
name: github-actions-mcp
event: pre-tool
matcher:
  tool: mcp__github__run_workflow|mcp__github__rerun_workflow_run|mcp__github__rerun_failed_jobs|mcp__github__cancel_workflow_run|mcp__plugin_github_github__run_workflow|mcp__plugin_github_github__rerun_workflow_run|mcp__plugin_github_github__rerun_failed_jobs|mcp__plugin_github_github__cancel_workflow_run
action:
  type: require-skill
  skill: github-actions
  on_miss: digest
  digest: digests/github-actions.md
summary: 'Before a prod workflow run or release, Claude checks the commit passed the earlier stage, says how to roll back, respects deploy windows, and never skips required checks.'
tags: [org, deploy]
---
Loads the github-actions skill before a GitHub MCP workflow dispatch, rerun or cancel.
