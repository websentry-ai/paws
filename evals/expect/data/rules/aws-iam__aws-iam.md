---
name: aws-iam
event: pre-tool
matcher:
  tool: Bash
  command_regex: ((^|[;&|\n])\s*|\$\()(\(\s*|(sudo\s+)?(\S*/)?(ba|z|da|k)?sh(\s+-\w+)*\s+-\w*c\w*\s+["']?)*\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((aws)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(iam\s+((create|attach|put|detach|delete|update|tag)-[a-z-]+|add-user-to-group))\b)
action:
  type: require-skill
  skill: aws-iam
tags: [org, aws]
---
Loads the aws-iam skill before any IAM write: users, roles, policies, access keys, attachments and tags.
