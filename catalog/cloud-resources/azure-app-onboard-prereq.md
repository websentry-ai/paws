---
name: azure-app-onboard-prereq
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '{**/.copilot-azure/sessions/**/prereq-output.json,**/.copilot-azure/sessions/**/readiness-report.md}'
action:
  type: require-skill
  skill: azure-app-onboard-prereq
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-app-onboard-prereq/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:5fdbc807950b48835e543167afda1f1ee310340567b117454df81690a449f6fc
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Statically checks a repo's build health, dependencies and stack fit for Azure, giving per-component verdicts.
tags:
- practice
- cloud-resources
---
Loads azure-app-onboard-prereq before the prereq output or readiness report is written, so verdicts follow its static-only rules.
