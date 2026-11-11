---
name: airunway-aks-setup
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?kubectl\s[^\n;&|]*\b(airunway|modeldeployments?|kaito)\b
action:
  type: require-skill
  skill: airunway-aks-setup
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/airunway-aks-setup/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:db59f2b53ac2c681024a8fff442297f95228c1fd06d054bd0cf21cff2a0dd014
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Walks an AKS cluster to a running model with AI Runway: controller, GPU check, providers, first deployment.'
tags:
- practice
- containers-orchestration
---
Loads airunway-aks-setup before kubectl touches AI Runway, KAITO or ModelDeployment resources, so the setup steps run in order.
