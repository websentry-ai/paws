---
name: entra-agent-id
event: pre-tool
matcher:
  tool: Bash
  command_regex: (?i)(^|[;&|\n(])\s*(sudo\s+)?((curl|wget)\s[^\n;&|]*graph\.microsoft\.com/\S*agentidentit|az\s+rest\s[^\n;&|]*graph\.microsoft\.com/\S*agentidentit|dotnet\s+add\b[^\n;&|]*package\s+Microsoft\.Identity\.Web\.AgentIdentities)
action:
  type: require-skill
  skill: entra-agent-id
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/entra-agent-id/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:60125cf89ebfeab7ba645ccfa86a5499857e141d6137cc360aa52f9c9df3fc5d
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Provisions Entra Agent Identity Blueprints and agent identities via Graph and sets up fmi_path/OBO token exchange.
tags:
- practice
- identity-access
---
Loads entra-agent-id before a Graph call to an agentIdentity endpoint or an AgentIdentities package install, so blueprint and token exchange steps are followed.
