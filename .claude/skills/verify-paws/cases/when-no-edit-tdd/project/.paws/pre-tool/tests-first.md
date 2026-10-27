---
name: tests-first
event: pre-tool
matcher:
  file_glob: "src/**"
when: no-edit-matching:tests/**
action:
  type: require-skill
  skill: tdd-probe
---
