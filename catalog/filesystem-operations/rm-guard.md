---
name: rm-guard
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(\w+=\S*\s+)*(sudo(\s+-\S+)*\s+)?rm\s
action:
  type: exec-script
  script: rm-guard.sh
tags: [org, safety, filesystem]
---
Refuses a recursive `rm` aimed at `/`, a top-level system directory, the home directory, `.`, `..`, a bare `*`, or a path that starts with a variable (`"$DIR/"` with an empty `$DIR` is `/`). Any other `rm` runs; the coreutils binding covers the list-before-delete habit.

This matches command text, not shell semantics: it stops an agent that is about to make a mistake, not one hiding the target.
