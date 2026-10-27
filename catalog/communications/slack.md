---
name: slack
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?curl\s[^;&|\n]*slack\.com/api/(chat\.(postMessage|scheduleMessage|update|delete)|files\.upload)
action:
  type: require-skill
  skill: slack
  on_miss: digest
  digest: digests/slack.md
tags: [org, comms]
---
Loads the slack skill before a message is posted, scheduled, edited, deleted or a file uploaded through the Slack Web API.
