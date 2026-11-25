---
name: "receiving-code-review-2"
event: "pre-tool"
matcher:
  tool: "mcp__*github*__get_pull_request_reviews|mcp__*github*__get_pull_request_comments"
action:
  type: require-skill
  skill: "receiving-code-review"
---
PR review comments are fetched
