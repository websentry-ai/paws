---
name: secret-scan-files
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: '{.env,.env.*,**/secrets/**}'
action:
  type: require-skill
  skill: secret-scan
  on_miss: digest
  digest: digests/secret-scan.md
tags: [org, secrets]
---
Loads secret-scan before a .env file or anything under secrets/ is written, so the file is ignored by git and holds no invented credentials.
