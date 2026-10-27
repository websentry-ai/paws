---
name: css-style
event: pre-tool
matcher:
  file_glob: "**/*.{css,scss}"
action:
  type: require-skill
  skill: frontend-style
---
