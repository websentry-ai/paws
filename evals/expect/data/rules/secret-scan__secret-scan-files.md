---
name: secret-scan-files
event: pre-tool
matcher:
  tool: "Write|Edit"
  file_glob: "{.env,.env.*,**/secrets/**}"
action:
  type: require-skill
  skill: secret-scan
---
A .env file or anything under secrets/ is written
