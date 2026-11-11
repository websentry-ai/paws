---
name: firebase-data-connect-files
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/dataconnect/**/{*.gql,dataconnect.yaml,connector.yaml}'
action:
  type: require-skill
  skill: firebase-data-connect
upstream:
  repo: github.com/firebase/agent-skills
  path: skills/firebase-data-connect-basics/SKILL.md
  ref: 99954a30898bb5ddc8152ee10e124dc3b4106bf7
  content_hash: sha256:b59a4e6876a1cd105a6e1463643f79d019d64bee8876f9a0d8d662f4f93d30b5
provenance:
  stars: 508
  license: Apache-2.0
  last_push: '2026-11-09'
  fetched: '2026-11-12'
tags:
- practice
- databases
---
Loads firebase-data-connect before Data Connect commands run or its schema and connector files are edited, so it follows the schema, authorization and SDK generation rules.
