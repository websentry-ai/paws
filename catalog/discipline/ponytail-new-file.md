---
name: ponytail-new-file
event: pre-tool
matcher:
  tool: Write
  file_glob: '*.{ts,tsx,js,jsx,mjs,cjs,go,py,rs,rb,java,kt,swift,c,cc,cpp,h,hpp,cs,php,vue,svelte}'
when: new-file
action:
  type: require-skill
  skill: ponytail
upstream:
  repo: github.com/DietrichGebert/ponytail
  path: skills/ponytail/SKILL.md
  ref: 09c434ab73ca43761c5d8b0c5d341926c2fba710
  content_hash: sha256:56d72677aca02689ac89bd48734e8c19b40ab86716d1257eaf5f1238b43cda6b
provenance:
  stars: 178631
  license: MIT
  last_push: 2026-10-18
  fetched: 2026-10-28
tags: [practice, discipline]
---
Loads ponytail before the agent creates a new source file, so it checks whether the file needs to exist and reuses what the codebase already has.
