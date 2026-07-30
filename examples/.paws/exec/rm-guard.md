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
# rm guard (exec-script escape hatch)
Delegates to a shell script that hard-denies any recursive force delete
(`rm -rf` / `rm -fr`), regardless of target — not just root/home paths.
