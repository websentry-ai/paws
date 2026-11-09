---
name: secret-scan-commit
event: pre-commit
matcher:
  tool: Bash
action:
  type: require-skill
  skill: secret-scan
  on_miss: digest
  digest: digests/secret-scan.md
summary: 'A secret check on staged changes before every commit, and .env and secrets/ files kept out of git.'
tags: [org, secrets, git]
---
Loads secret-scan before a commit, so the staged files and added lines are checked for credentials first.
