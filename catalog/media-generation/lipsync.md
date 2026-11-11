---
name: lipsync
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?@runcomfy/cli(@\S+)?|runcomfy)\s+(--output\s+\S+\s+)?run\s+((bytedance/omnihuman|sync/sync/lipsync|kling/lipsync)(/|($|[\s;&|)\'"]))|creatify/)
action:
  type: require-skill
  skill: lipsync
upstream:
  repo: github.com/prime-skills/runcomfy-agent-skills
  path: lipsync/SKILL.md
  ref: fca19ae084c2ae4667f5c8affbb99da5103d5832
  content_hash: sha256:05b6a828dc9f2fbaff4ed324864c6156a8b9bec7ac5f3be640c2ec0cbdf6520d
provenance:
  stars: 62
  license: MIT
  last_push: '2026-05-15'
  fetched: '2026-11-12'
summary: Lip-syncs a face to audio on RunComfy, routing to OmniHuman, Sync Labs, Kling lipsync or Creatify by input type.
tags:
- practice
- media-generation
---
Loads lipsync before a RunComfy lip-sync run, so the endpoint matching the input type (portrait or video) is used.
