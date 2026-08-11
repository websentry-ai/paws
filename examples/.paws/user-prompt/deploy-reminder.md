---
name: deploy-reminder
event: user-prompt
matcher:
  command_regex: (?i)\b(deploy|ship|release)
action:
  type: prompt-inject
  message: "paws: before deploying, confirm the change is behind a flag and the rollback path is known."
---
# deploy-reminder
Injects a checklist nudge on deploy/ship/release prompts.
