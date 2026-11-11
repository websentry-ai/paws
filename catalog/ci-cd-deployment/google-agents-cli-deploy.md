---
name: google-agents-cli-deploy
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(uvx\s+google-agents-cli|agents-cli)\s+(deploy|infra\s+cicd)\b
action:
  type: require-skill
  skill: google-agents-cli-deploy
upstream:
  repo: github.com/google/agents-cli
  path: skills/google-agents-cli-deploy/SKILL.md
  ref: 896e4719c4184841325a301755781d6b3da233ef
  content_hash: sha256:b04140ee79b7816141bd895c97b092743b4a4fd3094adfc960e55b2983bb4967
provenance:
  stars: 6337
  license: Apache-2.0
  last_push: '2026-11-04'
  fetched: '2026-11-12'
summary: Deploys agents to Agent Runtime, Cloud Run or GKE with agents-cli, plus CI/CD, secrets, service accounts and rollback.
tags:
- practice
- ci-cd-deployment
---
Loads google-agents-cli-deploy before agents-cli deploys or sets up CI/CD, so the target, secrets and service account are right.
