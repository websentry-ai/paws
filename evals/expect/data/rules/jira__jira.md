---
name: jira
event: pre-tool
matcher:
  tool: Bash
  command_regex: ((^|[;&|\n])\s*|\$\()(\(\s*|(sudo\s+)?(\S*/)?(ba|z|da|k)?sh(\s+-\w+)*\s+-\w*c\w*\s+["']?)*\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((jira)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(issue\s+(create|edit|move|delete|link|comment\s+add)|epic\s+(add|remove)|sprint\s+add)\b)
action:
  type: require-skill
  skill: jira
tags: [org, issues]
---
Loads the jira skill before Jira issues are created, edited, moved, linked, deleted or commented on.
