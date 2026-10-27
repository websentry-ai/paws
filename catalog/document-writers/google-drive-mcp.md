---
name: google-drive-mcp
event: pre-tool
matcher:
  tool: mcp__claude_ai_Google_Drive__create_file|mcp__claude_ai_Google_Drive__copy_file|mcp__google_drive__create_file|mcp__google_drive__copy_file|mcp__google-drive__create_file|mcp__google-drive__copy_file|mcp__gdrive__create_file|mcp__gdrive__copy_file
action:
  type: require-skill
  skill: google-drive
  on_miss: digest
  digest: digests/google-drive.md
tags: [org, docs]
---
Loads the google-drive skill before files are created or copied in Google Drive.
