---
name: azure-compliance-2
event: pre-tool
matcher:
  tool: mcp__*azure*__extension_azqr*
action:
  type: require-skill
  skill: azure-compliance
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-compliance/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:1030c28fe67f0ecb40ae30cf669e6bbac34c4362bc7f627489765f6f3f65f785
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Runs Azure compliance and security audits with azqr and checks Key Vault secret and certificate expiry.
tags:
- practice
- security-compliance
---
Loads azure-compliance before azqr is run, by CLI or MCP, so the scan, priority classification and Key Vault checks follow the audit workflow.
