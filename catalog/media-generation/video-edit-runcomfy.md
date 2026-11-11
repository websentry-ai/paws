---
name: video-edit-runcomfy
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npx\s+(-y\s+)?@runcomfy/cli|runcomfy)\s+run\s+(wan-ai/wan-2-7/edit-video|kling/kling-2-6/motion-control|decart/lucy-edit/)
action:
  type: require-skill
  skill: video-edit
upstream:
  repo: github.com/gencraft-labs/skills
  path: video-edit/SKILL.md
  ref: 5d625f59c710d7a33316b50b302b7158cdee5971
  content_hash: sha256:b666f441d10b6b387be2c88fd1dba9954b8e5baf179c8f3ee85984654e2f51db
provenance:
  stars: 3
  license: MIT
  last_push: '2026-10-06'
  fetched: '2026-11-12'
summary: Routes video edits (restyle, background swap, motion transfer) to Wan 2.7 Edit, Kling Motion Control or Lucy Edit.
tags:
- practice
- media-generation
---
Loads video-edit before a RunComfy video-edit model runs, so the call goes to the right model with its prompting pattern.
