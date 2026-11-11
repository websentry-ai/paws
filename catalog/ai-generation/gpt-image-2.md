---
name: gpt-image-2
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+openai/gpt-image-2/text-to-image($|[\s;&|)\'"])
action:
  type: require-skill
  skill: gpt-image-2
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: gpt-image-2/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:70cfab12b9f2e13a46f54976bd4f3eb7f3cc356a1e052d33241ca1b7a9d211c5
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Generates and edits images with GPT Image 2 on RunComfy through the runcomfy CLI.
tags:
- practice
- ai-generation
---
Loads gpt-image-2 before a RunComfy GPT Image 2 text-to-image run, so the fixed sizes and text-rendering tips are applied.
