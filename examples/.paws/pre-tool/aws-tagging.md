---
name: aws-tagging
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|])\s*(aws (ec2 run-instances|s3 mb)|terraform apply)
action:
  type: require-skill
  skill: aws-tagging-standards
  deny_message: Invoke the aws-tagging-standards skill for the required Owner/Environment/ManagedBy tags, then retry this exact command.
---
# aws-tagging
Requires the tagging skill before terraform apply / aws ec2|s3 create calls.
