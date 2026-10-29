---
name: fan-out-limit
event: pre-tool
matcher:
  tool: Agent|Task
when: agent-count>1
action:
  type: require-skill
  skill: fanout-review-7k
---
Past two subagents in one prompt, the fan-out notes load first.
