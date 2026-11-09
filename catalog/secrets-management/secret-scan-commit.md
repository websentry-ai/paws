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
summary: 'Before each commit, Claude checks staged files and lines for keys and tokens, and keeps .env and secrets/ files out of git. On a hit it unstages the file and tells you.'
tags: [org, secrets, git]
---
Loads secret-scan before a commit, so the staged files and added lines are checked for credentials first.
