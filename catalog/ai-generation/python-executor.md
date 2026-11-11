---
name: python-executor
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?belt\s+app\s+(run|sample)\s+\S*python-executor
action:
  type: require-skill
  skill: python-executor
upstream:
  repo: github.com/101-skills/superpowers
  path: tools/utilities/python-executor/SKILL.md
  ref: becc25649700d5457772a00e5143e28ccf9e5afa
  content_hash: sha256:83ea65ce7b0fe4384b2c1717a7cef07076924f4519538b189325f1951f37b889
provenance:
  stars: 23
  license: NOASSERTION
  last_push: '2026-08-31'
  fetched: '2026-11-12'
summary: Runs Python code in a hosted sandbox via the inference.sh python-executor app.
tags:
- practice
- ai-generation
---
Loads python-executor before running code on the hosted python-executor app, so it uses the right input schema and limits.
