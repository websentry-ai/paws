---
name: firebase-security-rules-auditor
event: pre-tool
matcher:
  tool: Read|Write|Edit
  file_glob: '**/{firestore,storage}.rules'
action:
  type: require-skill
  skill: firebase-security-rules-auditor
upstream:
  repo: github.com/firebase/agent-skills
  path: skills/firebase-security-rules-auditor/SKILL.md
  ref: 99954a30898bb5ddc8152ee10e124dc3b4106bf7
  content_hash: sha256:19bc79ffbc43d2bdbeef28d8f093e7f33748b8ba155d021f7efde2618d7544c1
provenance:
  stars: 508
  license: Apache-2.0
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Audits Firestore and Storage security rules for bypasses, escalation, missing type and size checks.
tags:
- practice
- identity-access
---
Loads firebase-security-rules-auditor before a Firestore or Storage rules file is read or edited, so it checks bypasses, escalation and missing type and size checks.
