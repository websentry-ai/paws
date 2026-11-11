---
name: python-appservice-deploy
event: pre-tool
matcher:
  tool: Bash
  command_regex: (?i)(^|[;&|\n(])\s*(sudo\s+)?az\s+webapp\s+(up|create)\b[^\n;&|]*python
action:
  type: require-skill
  skill: python-appservice-deploy
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/python-appservice-deploy/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:5f4fb493a45ae9ddbf416269ff13be06811f5f70b84e2c7f72616830bccffee0
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Deploys Flask, Django or FastAPI code to Azure App Service Linux, creating the resource group, plan and app if missing.
tags:
- practice
- ci-cd-deployment
---
Loads python-appservice-deploy before a Python web app is created or deployed with az webapp, so runtime, startup and build settings are right.
