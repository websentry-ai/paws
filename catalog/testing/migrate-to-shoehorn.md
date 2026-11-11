---
name: migrate-to-shoehorn
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npm|pnpm|yarn|bun)\s+(i|install|add)(\s+((--filter|-F|-w|--workspace|--prefix|-C|--dir|--cwd|--registry)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|-d|--dev|--exact|-W|--ignore-workspace-root-check|--legacy-peer-deps|--no-save|--ignore-scripts|[^\s;&|-][^\s;&|]*))*\s+@total-typescript/shoehorn(@\S+)?([\s;&|)]|$)
action:
  type: require-skill
  skill: migrate-to-shoehorn
upstream:
  repo: github.com/mattpocock/skills
  path: skills/misc/migrate-to-shoehorn/SKILL.md
  ref: 4ae2317ddb22b046e26455e4f9c9b3aaaf62212d
  content_hash: sha256:a63549df6d5b5c6595029f8c504ccf31a12f01312f03db70da4b45cb10ba5ca8
provenance:
  stars: 321219
  license: MIT
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Replaces `as` type assertions in tests with shoehorn's fromPartial, fromAny and fromExact helpers.
tags:
- practice
- testing
---
Loads migrate-to-shoehorn before @total-typescript/shoehorn is installed, so test-only usage and the as-to-fromPartial patterns are followed.
