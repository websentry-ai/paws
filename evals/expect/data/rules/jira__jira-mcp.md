---
name: jira-mcp
event: pre-tool
matcher:
  tool: mcp__atlassian__createJiraIssue|mcp__atlassian__editJiraIssue|mcp__atlassian__transitionJiraIssue|mcp__atlassian__addCommentToJiraIssue
action:
  type: require-skill
  skill: jira
summary: 'Clean Jira tickets: search for duplicates first, follow your template, and link each issue to its epic.'
tags: [org, issues]
---
Loads the jira skill before an Atlassian MCP issue create, edit, transition or comment.
