---
name: infra-commit
event: pre-commit
matcher:
  tool: Bash
when: repo-marker:infra/terraform
action:
  type: require-skill
  skill: ledger-notes
---
Repos with Terraform under infra/ follow the ledger-notes conventions.
