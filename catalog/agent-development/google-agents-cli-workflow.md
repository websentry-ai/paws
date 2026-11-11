---
name: google-agents-cli-workflow
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/.agents-cli-spec.md'
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
tags:
- practice
- agent-development
---
Loads google-agents-cli-workflow before the spec is written or agents-cli is first set up, so the lifecycle and skill order are known.
