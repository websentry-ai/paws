---
name: ticket-ref
event: pre-commit
matcher:
  tool: Bash
action:
  type: exec-script
  script: ticket-ref.sh
---
Commit messages must reference a PAWS-<n> ticket.
