---
name: entra-app-registration
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?az\s+ad\s+(app\s+(create|update|credential|permission|federated-credential|owner)|sp\s+(create|create-for-rbac|update|credential|owner))\b
action:
  type: require-skill
  skill: entra-app-registration
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/entra-app-registration/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:275936344b67fe14a0c4ec6977a1c5dc45d145572280017dd328a32f01776c96
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Guides Entra ID app registration, OAuth 2.0 setup, API permissions, service principals and MSAL integration.
tags:
- practice
- identity-access
---
Loads entra-app-registration before an app registration or service principal is created or changed, so auth type, permissions and IaC advice apply.
