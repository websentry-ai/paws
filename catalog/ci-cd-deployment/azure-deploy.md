---
name: azure-deploy
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(azd\s+(up|deploy|provision)([\s;&|)]|$)|az\s+deployment\s+(group|sub|mg|tenant)\s+create\b)
action:
  type: require-skill
  skill: azure-deploy
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-deploy/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:6c12c077960a3470ef789a64aa04a5c86f146e23fbd0262c6ffc1980c5e418eb
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Deploys an already prepared and validated Azure app with azd up or deploy, Bicep or Terraform, with recovery steps.
tags:
- practice
- ci-cd-deployment
---
Loads azure-deploy before azd up, azd deploy or an az deployment create runs, so the prepared plan and Validated status are checked first.
