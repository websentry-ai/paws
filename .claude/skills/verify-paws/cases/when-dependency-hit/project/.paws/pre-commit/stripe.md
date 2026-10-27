---
name: stripe-commit
event: pre-commit
matcher:
  tool: Bash
when: dependency:stripe
action:
  type: require-skill
  skill: payments-review
---
