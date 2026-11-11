---
name: product-marketing
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/{.agents,.claude}/product-marketing*.md'
action:
  type: require-skill
  skill: product-marketing
upstream:
  repo: github.com/coreyhaines31/marketingskills
  path: skills/product-marketing/SKILL.md
  ref: 3e729622e3de0c82abdb906190d91f9018045872
  content_hash: sha256:2b7f80ab415280e3e0a0bcc112af8e24059169642c70fb36e18e44a79f80f505
provenance:
  stars: 59621
  license: MIT
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: Creates and maintains the product marketing context doc (product, ICP, positioning, voice) other marketing skills read.
tags:
- practice
- marketing
---
Loads product-marketing before the product marketing context doc is written or edited, so it keeps the doc's sections, version and changelog consistent.
