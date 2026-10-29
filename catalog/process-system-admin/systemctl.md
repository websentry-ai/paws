---
name: systemctl
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?((systemctl)(\s+-\S+(\s+[^-\s]\S*)?)*\s+(start|stop|restart|reload|enable|disable|mask|edit|daemon-reload|kill)\b|crontab(\s+-u\s+\S+)?\s+(-[a-km-zA-Z]|-(\s|$))|kill\s+(-9|-KILL|-SIGKILL|-s\s+(KILL|9))\b|(pkill|killall)\b)
action:
  type: require-skill
  skill: systemctl
  on_miss: digest
  digest: digests/systemctl.md
tags: [org, system]
---
Loads the systemctl skill before services are started, stopped, restarted, enabled or edited, crontabs change, or processes are force-killed or killed by pattern.
