---
name: aws-tagging
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|])\s*(aws (ec2 run-instances|s3 mb)|terraform apply)
action:
  type: require-skill
  skill: aws-tagging-standards
  deny_message: Load the aws-tagging-standards skill (read its SKILL.md) for the required Owner/Environment/ManagedBy tags — paws still denies an automated retry of this exact command; a human needs to run it directly.
---
# AWS Tagging Hook

Ensures every AWS resource created through the agent carries the standard
Owner / Environment / ManagedBy tags before it is provisioned.
