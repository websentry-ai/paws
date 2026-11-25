---
name: "receiving-code-review-3"
event: "pre-tool"
matcher:
  tool: "mcp__*github*__pull_request_read"
  command_regex: "\"method\"\\s*:\\s*\"get_(review_comments|reviews|comments)\""
action:
  type: require-skill
  skill: "receiving-code-review"
---
PR review comments are fetched
