---
name: firecrawl
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(firecrawl|(npx\s+(-y\s+)?|bunx\s+)firecrawl-cli(@\S+)?)\s+[a-z]
action:
  type: require-skill
  skill: firecrawl
upstream:
  repo: github.com/firecrawl/cli
  path: skills/firecrawl/SKILL.md
  ref: 06b371f06c9798f50a8a64f87a88dd34f36f91e7
  content_hash: sha256:a49e835d3121cf35bcd1db11199e58a1a4f3c11e219311204274063dad048a1a
provenance:
  stars: 683
  license: NOASSERTION
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: Searches, scrapes, maps and crawls the live web with the Firecrawl CLI, saving clean markdown.
tags:
- practice
- browser
---
Loads firecrawl before the Firecrawl CLI is run to search or scrape the web, so it uses the right commands, output files and security rules.
