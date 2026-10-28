---
name: fan-out-limit
event: pre-tool
matcher:
  tool: Agent|Task
when: agent-count>1
action:
  type: block
  deny_message: at most two subagents per prompt in this repo.
---
Caps subagent fan-out at two per prompt.
