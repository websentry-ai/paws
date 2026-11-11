---
name: azure-app-onboard
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '{**/.copilot-azure/sessions/**/context.json,**/.copilot-azure/sessions/**/scaffold-manifest.json,**/.copilot-azure/sessions/**/deploy-checklist.md}'
action:
  type: require-skill
  skill: azure-app-onboard
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-app-onboard/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:4643f25fd60740ed5ce40e4baa271964fa1fc2b11368c76955fae525bc9253ac
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Runs an idea-or-repo to Azure pipeline: analyze, pick services, estimate cost, scaffold IaC, deploy after approval.'
tags:
- practice
- cloud-resources
---
Loads azure-app-onboard before its session context, scaffold manifest or deploy checklist is written, so the pipeline gates are kept.
