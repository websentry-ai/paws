---
name: azure-kubernetes
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?az\s+aks\s+(create|enable-addons|nodepool\s+add)\b
action:
  type: require-skill
  skill: azure-kubernetes
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/azure-kubernetes/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:e75aa0b540f3eaaf95cfe2f659f8cb1eb2fba9b04a0bc62ac81a002d5537c6c6
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Plans and creates production AKS clusters: Day-0 networking, SKU, security, autoscaling and upgrade choices.'
tags:
- practice
- containers-orchestration
---
Loads azure-kubernetes before an AKS cluster or node pool is created, so Day-0 networking, SKU and security choices are made deliberately.
