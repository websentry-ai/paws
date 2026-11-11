---
name: ai-seo
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/llms{,-full}.txt'
action:
  type: require-skill
  skill: ai-seo
upstream:
  repo: github.com/coreyhaines31/marketingskills
  path: skills/ai-seo/SKILL.md
  ref: 3e729622e3de0c82abdb906190d91f9018045872
  content_hash: sha256:5b5dbb143389784211c56e357e20bd44a12c3a0245dbf6dcfbb40348daaaf4ef
provenance:
  stars: 59621
  license: MIT
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: 'Improves the chance of being cited by AI answer engines: content formats, llms.txt, agent-readable sites.'
tags:
- practice
- marketing
---
Loads ai-seo before llms.txt or llms-full.txt is written or edited, so it follows the agent-readable site guidance.
