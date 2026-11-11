---
name: google-agents-cli-eval
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(uvx\s+google-agents-cli|agents-cli)\s+eval\b
action:
  type: require-skill
  skill: google-agents-cli-eval
upstream:
  repo: github.com/google/agents-cli
  path: skills/google-agents-cli-eval/SKILL.md
  ref: 896e4719c4184841325a301755781d6b3da233ef
  content_hash: sha256:46239dcdfc6615c09258125d2358561b85cfc9d7c6d38f9c6be1a5b42d613172
provenance:
  stars: 6337
  license: Apache-2.0
  last_push: '2026-11-04'
  fetched: '2026-11-12'
summary: 'Runs and analyses agent evaluations with agents-cli: datasets, metrics, LLM-as-judge and the eval-fix loop.'
tags:
- practice
- testing
---
Loads google-agents-cli-eval before agents-cli eval runs, so the dataset schema and eval-fix loop are followed.
