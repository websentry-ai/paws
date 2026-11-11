---
name: infsh-cli
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(belt|infsh)\s+(app\s+(list|search|get|sample)|task\s+get)($|\s)|(^|[;&|\n(])\s*(sudo\s+)?(belt|infsh)\s+app\s+run\s+(openrouter|infsh)/|(^|[;&|\n(])\s*(sudo\s+)?curl\s[^;&|\n]*https://cli\.inference\.sh
action:
  type: require-skill
  skill: infsh-cli
upstream:
  repo: github.com/101-skills/superpowers
  path: tools/infsh-cli/SKILL.md
  ref: becc25649700d5457772a00e5143e28ccf9e5afa
  content_hash: sha256:e6e21f4e0e9bf135b4edd3e5e0e4263051fc8715173cd647eac42a5cfadeab1c
provenance:
  stars: 23
  license: NOASSERTION
  last_push: '2026-08-31'
  fetched: '2026-11-12'
summary: Runs hosted AI apps (image, video, LLM, search) through the inference.sh belt CLI.
tags:
- practice
- ai-generation
---
Loads infsh-cli before browsing the inference.sh app store, running a non-specialised app, or installing the belt CLI, so it uses the right belt commands.
