---
name: jira-mcp
event: pre-tool
matcher:
  tool: mcp__atlassian__createJiraIssue|mcp__atlassian__editJiraIssue|mcp__atlassian__transitionJiraIssue|mcp__atlassian__addCommentToJiraIssue
action:
  type: require-skill
  skill: jira
  on_miss: digest
  digest: digests/jira.md
tags: [org, issues]
---
Loads the jira skill before an Atlassian MCP issue create, edit, transition or comment.
