---
name: remotion-docs
event: pre-tool
matcher:
  tool: WebFetch
  command_regex: ^https?://([\w-]+\.)*remotion\.dev/docs($|[/?#])
action:
  type: require-skill
  skill: remotion-docs
upstream:
  repo: github.com/remotion-dev/skills
  path: skills/remotion-docs/SKILL.md
  ref: 1af7e69a2a613437d46095924057970a6a7df40d
  content_hash: sha256:1c993ff9a8f827d586578c4d575ead197ddb708ea49f981f1aa52934569ceee2
provenance:
  stars: 6813
  license: NOASSERTION
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Searches the Remotion docs via Algolia and fetches pages as Markdown instead of relying on memory.
tags:
- practice
- video-generation
---
Loads remotion-docs before a Remotion docs page is fetched, so the Algolia search and Markdown fetch method are used.
