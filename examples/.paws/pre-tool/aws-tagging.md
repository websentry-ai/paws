---
name: aws-tagging
event: pre-tool
matcher:
  tool: Bash
  command_regex: ^(aws (ec2 run-instances|s3 mb)|terraform apply)
action:
  type: require-skill
  skill: aws-tagging-standards
  deny_message: Load the aws-tagging-standards skill (read its SKILL.md), then retry the exact same command. Every AWS resource must carry Owner/Environment/ManagedBy tags.
---
# AWS Tagging Hook

Ensures every AWS resource created through the agent carries the standard
Owner / Environment / ManagedBy tags before it is provisioned.
