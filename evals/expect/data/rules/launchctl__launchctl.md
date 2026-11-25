---
name: "launchctl"
event: "pre-tool"
matcher:
  tool: "Bash"
  command_regex: "((^|[;&|\\n])\\s*|\\$\\()(\\(\\s*|(sudo\\s+)?(\\S*/)?(ba|z|da|k)?sh(\\s+-\\w+)*\\s+-\\w*c\\w*\\s+[\"']?)*\\s*(\\w+=\\S*\\s+)*(sudo(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+)?((launchctl)(\\s+-\\S+(\\s+[^-\\s]\\S*)?)*\\s+(bootstrap|bootout|load|unload|enable|disable|kickstart|kill)\\b)"
action:
  type: require-skill
  skill: "launchctl"
---
Launchd jobs are loaded, unloaded, enabled, disabled, kicked or killed
