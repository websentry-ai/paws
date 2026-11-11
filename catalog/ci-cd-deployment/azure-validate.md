---
name: azure-validate
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(az\s+deployment\s+\S+\s+(what-if|validate)\b|azd\s+provision\b[^;&|\n]*--preview\b)
action:
  type: require-skill
  skill: azure-validate
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-validate/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:0cb906621911266d762ba55f897cf64e0ea08c4369e7d919faf3a62da84bf6c5
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Runs pre-deployment checks on Azure config, Bicep or Terraform, RBAC and identities, then marks the plan Validated.
tags:
- practice
- ci-cd-deployment
---
Loads azure-validate before an Azure what-if, validate or provision preview runs, so the validation workflow runs before any deployment.
