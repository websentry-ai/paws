---
name: gsutil
event: pre-tool
matcher:
  tool: Bash
  command_regex: ((^|[;&|\n])\s*|\$\()(\(\s*|(sudo\s+)?(\S*/)?(ba|z|da|k)?sh(\s+-\w+)*\s+-\w*c\w*\s+["']?)*\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((gsutil)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(cp|rsync|mv|rm)\b|(gcloud)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(storage\s+(cp|rsync|rm))\b)
action:
  type: require-skill
  skill: gsutil
summary: 'Safe Cloud Storage transfers: size big copies first, dry-run syncs, and copy only to buckets your org owns.'
tags: [org, cloud]
---
Loads the gsutil skill before objects are copied, synced, moved or removed in Google Cloud Storage.
