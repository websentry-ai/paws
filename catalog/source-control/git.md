---
name: git
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((git)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(commit|push|cherry-pick)([\s;&|)]|$)|(git)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(merge|rebase)(\s+(-[^-\s]|--([b-pr-z]|a[^b]|q[^u]|qu[^i]|qui[^t])|[^-\s;&|])|\s*($|[;&|)]))|(git)(\s+-\S+(\s+[^-\s]\S*)?)*\s+tag\s+(-[a-zA-Z]*[adsfmu]|--(annotate|sign|delete|force|message)|[^-\s;&|])|(gh)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(pr\s+(create|merge))\b)
action:
  type: require-skill
  skill: git
  on_miss: digest
  digest: digests/git.md
tags: [org, git]
---
Loads the git skill before commits, pushes, merges, rebases, tags, cherry-picks, and PR create or merge.
