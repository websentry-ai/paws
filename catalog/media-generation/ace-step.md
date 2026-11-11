---
name: ace-step
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+acestep-ai/ace-step(-1\.5)?/
action:
  type: require-skill
  skill: ace-step
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: ace-step/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:55506bcec721cc313f370be74d1ee7da441abd462c05f13099a9cdb9144971fd
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Generates, inpaints and extends music with ACE Step on RunComfy using tag-driven prompts and section-marked lyrics.
tags:
- practice
- media-generation
---
Loads ace-step before a RunComfy ACE Step run, so tag-driven prompts and section-marked lyrics are used.
