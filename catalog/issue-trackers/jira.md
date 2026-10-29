---
name: jira
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((jira)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(issue\s+(create|edit|move|delete|link|comment\s+add)|epic\s+(add|remove)|sprint\s+add)\b)
action:
  type: require-skill
  skill: jira
  on_miss: digest
  digest: digests/jira.md
tags: [org, issues]
---
Loads the jira skill before Jira issues are created, edited, moved, linked, deleted or commented on.
