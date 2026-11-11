---
name: azure-prepare-2
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?azd\s+init(\s|$)
action:
  type: require-skill
  skill: azure-prepare
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-prepare/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:18db180c7764f1a2d28840175194ab04c93ac133a00f6ea4c723c16b6bf90bea
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Prepares an app for azd deployment: writes azure.yaml, Bicep or Terraform, Dockerfiles and a deployment plan.'
tags:
- practice
- infrastructure-as-code
---
Loads azure-prepare before the Azure deployment plan is written or azd init runs, so planning, approval and generated infra follow its rules.
