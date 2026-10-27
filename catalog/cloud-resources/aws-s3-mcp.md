---
name: aws-s3-mcp
event: pre-tool
matcher:
  tool: mcp__aws-api__s3_create_bucket|mcp__aws-api__s3_put_bucket_policy
action:
  type: require-skill
  skill: aws-s3
  on_miss: digest
  digest: digests/aws-s3.md
tags: [org, aws]
---
Loads the aws-s3 skill before an AWS MCP server creates a bucket or writes a bucket policy.
