---
name: new-module
event: pre-tool
matcher:
  tool: Write|Edit
when: new-file
action:
  type: require-skill
  skill: module-banner
---
New modules follow the module-banner conventions.
