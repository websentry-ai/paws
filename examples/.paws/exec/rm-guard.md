---
name: rm-guard
event: pre-tool
matcher:
  tool: Bash
  command_regex: \brm\b
action:
  type: exec-script
  script: rm-guard.sh
---
# rm-guard
Denies `rm -rf`/`rm -fr` via a delegated script, any target.
