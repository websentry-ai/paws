---
name: aws-s3
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((aws)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(s3api\s+(create-bucket|put-bucket-policy|put-bucket-acl|delete-public-access-block|put-public-access-block)|s3\s+mb)\b)
action:
  type: require-skill
  skill: aws-s3
  on_miss: digest
  digest: digests/aws-s3.md
tags: [org, aws]
---
Loads the aws-s3 skill before an S3 bucket is created or its policy, ACL or public-access block changes.
