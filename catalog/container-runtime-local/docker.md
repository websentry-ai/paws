---
name: docker
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((docker)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(run|create|compose\s+(up|down)|rm|rmi|volume\s+(rm|prune)|system\s+prune|network\s+create|buildx?|exec|push)\b|(podman)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(run|rm|volume)\b)
action:
  type: require-skill
  skill: docker
  on_miss: digest
  digest: digests/docker.md
tags: [org, containers]
---
Loads the docker skill before local containers are run, built, exec'd, pushed or removed, and before volumes or networks change.
