---
name: ssh
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?(ssh(\s+(-[AaCfGgKkMNnqsTtVvXxYy46]+|-[BbcDEeFIiJLlmOoPpQRSWw]\s*[^\s;&|]+))*\s+(-o\s*[^\s;&|]*(StrictHostKeyChecking|UserKnownHostsFile=/dev/null)|-[a-zA-Z46]*[ALRD](\s|\d|$))|(scp|sftp)\s|ssh-keygen\s[^;&|\n]*-R\b)
action:
  type: require-skill
  skill: ssh
  on_miss: digest
  digest: digests/ssh.md
tags: [org, network]
---
Loads the ssh skill before host-key overrides, agent or port forwarding, scp/sftp transfers and known_hosts removals.
