---
name: gpt-image-edit
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+openai/gpt-image-2/edit($|[\s;&|)\'"])
action:
  type: require-skill
  skill: gpt-image-edit
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: gpt-image-edit/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:c2b7c34cec759141b6c012953920cb18fd8709f05834e2b62a8240dee7c03ab7
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: 'Edits images with GPT Image 2 /edit on RunComfy: identity-preserving edits, text rewrites, multi-reference input.'
tags:
- practice
- media-generation
---
Loads gpt-image-edit before a RunComfy GPT Image 2 edit run, so preservation language and text-rewrite patterns are used.
