---
name: companion-docs
event: pre-push
action:
  type: warn
  message: "paws: pushing to remote — remember the docs-companion. If you changed public APIs, update the docs before this lands."
---
# Companion Docs (pre-push)

Husky-flavored synthetic event: matches a Bash `git push` at runtime.
Warns (does not block) so the agent is reminded to update docs on push.
