---
name: aws-s3-call-aws-mcp
event: pre-tool
matcher:
  tool: mcp__aws-api__call_aws
  command_regex: \baws\s+(s3api\s+(create-bucket|put-bucket-policy|put-bucket-acl|delete-public-access-block|put-public-access-block)|s3\s+mb)\b
action:
  type: require-skill
  skill: aws-s3
  on_miss: digest
  digest: digests/aws-s3.md
summary: 'New S3 buckets get Block Public Access, default encryption, your org tags and an expiry tag. Claude states any bucket policy or ACL change before making it.'
tags: [org, aws]
---
Loads the aws-s3 skill when the generic call_aws MCP tool runs an S3 bucket create or policy change.
