---
name: analytics
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npm\s+(i|install|add)|pnpm\s+(i|install|add)|yarn\s+(global\s+)?add|bun\s+(i|install|add))(\s+((--filter|-F|-w|--workspace|--prefix|-C|--dir|--cwd|--registry)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|-d|--dev|--exact|-W|--ignore-workspace-root-check|--legacy-peer-deps|--no-save|--ignore-scripts|[^\s;&|-][^\s;&|]*))*\s+(posthog-js|posthog-node|mixpanel|mixpanel-browser|@segment/analytics-next|@segment/analytics-node|react-ga4|@amplitude/analytics-browser|@amplitude/analytics-node)(@[^\s;&|)]*)?(\s|$|[;&|)])
action:
  type: require-skill
  skill: analytics
upstream:
  repo: github.com/coreyhaines31/marketingskills
  path: skills/analytics/SKILL.md
  ref: 3e729622e3de0c82abdb906190d91f9018045872
  content_hash: sha256:064ca10386885a91dc0a7251fe6993606df4e37344d620370d8744a26c970938
provenance:
  stars: 59621
  license: MIT
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: 'Plans and audits analytics tracking: event naming, tracking plans, GA4, GTM, Mixpanel, PostHog, UTMs.'
tags:
- practice
- data-analytics
---
Loads analytics before an analytics SDK is installed, so events are named and planned before tracking code is added.
