---
name: wan-2-7
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+wan-ai/wan-2-7/text-to-video($|[\s;&|)\'"])
action:
  type: require-skill
  skill: wan-2-7
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: wan-2-7/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:cbca31f6b628ecf253440234f8d720fbd9b92da302c09206a56add406a381080
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: 'Generates video with Wan 2.7 text-to-video on RunComfy: multi-reference conditioning, audio lip-sync, duration schema.'
tags:
- practice
- media-generation
---
Loads wan-2-7 before a RunComfy Wan 2.7 text-to-video run, so the duration schema and reference options are right.
