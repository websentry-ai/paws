---
name: "ssh"
event: "pre-tool"
matcher:
  tool: "Bash"
  command_regex: "((^|[;&|\\n])\\s*|\\$\\()(\\(\\s*|(sudo\\s+)?(\\S*/)?(ba|z|da|k)?sh(\\s+-\\w+)*\\s+-\\w*c\\w*\\s+[\"']?)*\\s*(\\w+=\\S*\\s+)*(sudo(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+)?(ssh(\\s+(-[AaCfGgKkMNnqsTtVvXxYy46]+|-[BbcDEeFIiJLlmOoPpQRSWw]\\s*[^\\s;&|]+))*\\s+(-o\\s*[^\\s;&|]*(StrictHostKeyChecking|UserKnownHostsFile=/dev/null)|-[a-zA-Z46]*[ALRD](\\s|\\d|$))|(scp|sftp)\\s|ssh-keygen\\s[^;&|\\n]*-R\\b)"
action:
  type: require-skill
  skill: "ssh"
---
Host-key overrides, agent or port forwarding
