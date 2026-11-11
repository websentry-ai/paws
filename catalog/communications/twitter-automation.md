---
name: twitter-automation
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?belt\s+app\s+(run|sample)\s+x/
action:
  type: require-skill
  skill: twitter-automation
upstream:
  repo: github.com/101-skills/superpowers
  path: tools/social/twitter-automation/SKILL.md
  ref: becc25649700d5457772a00e5143e28ccf9e5afa
  content_hash: sha256:d658f8da511d87002505f1592c386cd8882fc2d648b57efdd1838888479a78a3
provenance:
  stars: 23
  license: NOASSERTION
  last_push: '2026-08-31'
  fetched: '2026-11-12'
summary: Posts tweets, likes, retweets, sends DMs and follows users on X through the inference.sh belt CLI.
tags:
- practice
- communications
---
Loads twitter-automation before posting or engaging on X through the belt CLI, so it uses the right x/ apps and does not delete or spam by accident.
