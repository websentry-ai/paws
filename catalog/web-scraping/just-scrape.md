---
name: just-scrape
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+)?)?just-scrape(@\S+)?\s+(search|scrape|extract|crawl|monitor|history|credits|validate)($|[^\w-])
action:
  type: require-skill
  skill: just-scrape
upstream:
  repo: github.com/scrapegraphai/just-scrape
  path: skills/just-scrape/SKILL.md
  ref: a3d179ffe2695c5bb7aacffec2c6018d3d9f056e
  content_hash: sha256:3404017710353002efad656fb640a7084f4a5f3bf53ff0565509be5c6916fb5f
provenance:
  stars: 76
  license: MIT
  last_push: '2026-09-21'
  fetched: '2026-11-12'
summary: Searches, scrapes, crawls, extracts JSON and monitors pages with the ScrapeGraph AI just-scrape CLI.
tags:
- practice
- web-scraping
---
Loads just-scrape before the ScrapeGraph CLI runs, so the search, scrape, extract and crawl flags are used correctly.
