---
name: companion-docs
event: pre-push
action:
  type: audit
---
# companion-docs
Logs every git push to `~/.paws/audit.log` — does not block, agent sees nothing.
