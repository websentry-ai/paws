---
name: google-agents-cli-workflow-2
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(uvx\s+google-agents-cli\s+setup\b|agents-cli\s+(setup|info|playground)\b)
action:
  type: require-skill
  skill: google-agents-cli-workflow
upstream:
  repo: github.com/google/agents-cli
  path: skills/google-agents-cli-workflow/SKILL.md
  ref: 896e4719c4184841325a301755781d6b3da233ef
  content_hash: sha256:fa5ec0124f848e7ea57efbaa4440515eb02bf3af428e06de65d67b7f03f935d7
provenance:
  stars: 6337
  license: Apache-2.0
  last_push: '2026-11-04'
  fetched: '2026-11-12'
summary: Lays out the ADK agent lifecycle (spec, scaffold, build, eval, deploy, publish, observe) and when to load each skill.
tags:
- practice
- agent-development
---
Loads google-agents-cli-workflow before the spec is written or agents-cli is first set up, so the lifecycle and skill order are known.
