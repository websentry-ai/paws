---
name: ads
event: pre-tool
matcher:
  tool: mcp__*google-ads__*|mcp__*google_ads__*|mcp__*googleads__*|mcp__*meta-ads__*|mcp__*meta_ads__*|mcp__*facebook-ads__*|mcp__*facebook_ads__*|mcp__*linkedin-ads__*|mcp__*linkedin_ads__*|mcp__*Google_Ads__*|mcp__*Meta_Ads__*|mcp__*LinkedIn_Ads__*
action:
  type: require-skill
  skill: ads
upstream:
  repo: github.com/coreyhaines31/marketingskills
  path: skills/ads/SKILL.md
  ref: 3e729622e3de0c82abdb906190d91f9018045872
  content_hash: sha256:bcda04d2c659ed107b6eaec71c0e39cdb3c7de952db50007b9240e8b8281b773
provenance:
  stars: 59621
  license: MIT
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: Plans, runs and fixes paid ads on Google, Meta and LinkedIn, separating facts from hypotheses and missing data.
tags:
- practice
- marketing
---
Loads ads before an ad platform MCP tool is called, so it separates facts from hypotheses and labels missing data.
