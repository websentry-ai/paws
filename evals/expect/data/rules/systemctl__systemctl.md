---
name: "systemctl"
event: "pre-tool"
matcher:
  tool: "Bash"
  command_regex: "((^|[;&|\\n])\\s*|\\$\\()(\\(\\s*|(sudo\\s+)?(\\S*/)?(ba|z|da|k)?sh(\\s+-\\w+)*\\s+-\\w*c\\w*\\s+[\"']?)*\\s*(\\w+=\\S*\\s+)*(sudo(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+)?((systemctl)(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+(start|stop|restart|reload|enable|disable|mask|edit|daemon-reload|kill)\\b|crontab(\\s+-u\\s+\\S+)?\\s+(-[a-km-zA-Z]|-(\\s|$))|kill\\s+(-9|-KILL|-SIGKILL|-s\\s+(KILL|9))\\b|(pkill|killall)\\b)"
action:
  type: require-skill
  skill: "systemctl"
---
Services are started, stopped, restarted, enabled or edited, crontabs change
