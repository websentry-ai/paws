---
name: "receiving-code-review"
event: "pre-tool"
matcher:
  tool: "Bash"
  command_regex: "((^|[;&|\\n])\\s*|\\$\\()(\\(\\s*|(sudo\\s+)?(\\S*/)?(ba|z|da|k)?sh(\\s+-\\w+)*\\s+-\\w*c\\w*\\s+[\"']?)*\\s*(sudo\\s+)?gh\\s+(pr\\s+view\\b[^;&|\\n]*--comments|api\\s+[\"\\']?repos/\\S+/pulls/\\d+/(comments|reviews)[\"\\']?(\\s+--paginate)?(\\s+(--jq|-q)\\s+(\\'[^\\']*\\'|\"[^\"]*\"|\\S+))*\\s*($|[;&|\\n)]))"
action:
  type: require-skill
  skill: "receiving-code-review"
---
PR review comments are fetched
