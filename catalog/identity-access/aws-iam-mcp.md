---
name: aws-iam-mcp
event: pre-tool
matcher:
  tool: mcp__aws-api__call_aws
  command_regex: \baws\s+iam\s+((create|attach|put|detach|delete|update|tag)-[a-z-]+|add-user-to-group)\b
action:
  type: require-skill
  skill: aws-iam
  on_miss: digest
  digest: digests/aws-iam.md
summary: 'Claude checks IAM policies for wildcards, diffs a policy against its current version, prefers roles to users, gives access keys an expiry, and never widens its own access.'
tags: [org, aws]
---
Loads the aws-iam skill when the generic call_aws MCP tool runs an IAM write.
