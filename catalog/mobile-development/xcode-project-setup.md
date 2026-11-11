---
name: xcode-project-setup
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '**/*.pbxproj'
action:
  type: require-skill
  skill: xcode-project-setup
upstream:
  repo: github.com/firebase/agent-skills
  path: skills/xcode-project-setup/SKILL.md
  ref: 99954a30898bb5ddc8152ee10e124dc3b4106bf7
  content_hash: sha256:4787904efddcb2deda6b67000f97ce0182e3ecf323aacb020329f495b6208d48
provenance:
  stars: 508
  license: Apache-2.0
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Safely edits Xcode projects to add Swift packages and link files using a script instead of hand edits.
tags:
- practice
- mobile-development
---
Loads xcode-project-setup before an Xcode project.pbxproj file is edited, so packages and files are added with the script instead of by hand.
