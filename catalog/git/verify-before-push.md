---
name: verify-before-push
event: pre-push
matcher:
  tool: Bash
action:
  type: require-skill
  skill: verification-before-completion
upstream:
  repo: github.com/obra/superpowers
  path: skills/verification-before-completion/SKILL.md
  ref: 2ce8aa15d05bd57213e6f746f953aafb7ded7a8b
  content_hash: sha256:09b0e3a4689c1c38a3b8fc6ef57a1d4165aa33c0013d546dbde9dd348941a319
provenance:
  stars: 305962
  license: MIT
  last_push: 2026-10-23
  fetched: 2026-10-27
tags: [practice, git]
---
Loads verification-before-completion before a push, so nothing leaves the machine on an unverified claim that it works.
