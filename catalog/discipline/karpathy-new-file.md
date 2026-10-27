---
name: karpathy-new-file
event: pre-tool
matcher:
  tool: Write
  file_glob: '*.{ts,tsx,js,jsx,mjs,cjs,go,py,rs,rb,java,kt,swift,c,cc,cpp,h,hpp,cs,php,vue,svelte}'
when: new-file
action:
  type: require-skill
  skill: karpathy-guidelines
upstream:
  repo: github.com/multica-ai/andrej-karpathy-skills
  path: skills/karpathy-guidelines/SKILL.md
  ref: 2c606141936f1eeef17fa3043a72095b4765b9c2
  content_hash: sha256:e64131803b0a34b8c39afc11338a74f8305cf8f69f750620bdf6cb38f6865c52
provenance:
  stars: 225218
  license: MIT
  last_push: 2026-04-20
  fetched: 2026-10-27
tags: [practice, discipline]
---
Loads karpathy-guidelines before the agent creates a new source file, the moment overbuilding usually starts.
