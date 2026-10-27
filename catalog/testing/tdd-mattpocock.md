---
name: tdd-mattpocock
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: 'src/**/*.{ts,tsx,js,jsx,mjs,cjs,go,py,rs,rb,java,kt,swift,c,cc,cpp,h,hpp,cs,php}'
when: 'no-edit-matching:{*_test.*,*.test.*,*.spec.*,test_*,*_spec.*,__tests__/**,tests/**,test/**,spec/**}'
action:
  type: require-skill
  skill: tdd
upstream:
  repo: github.com/mattpocock/skills
  path: skills/engineering/tdd/SKILL.md
  ref: 59db609eb1e18aaf59b63284d0e52255aff0c63c
  content_hash: sha256:e3e0f78ff1c2d52ffefa0378baa1e80e129d359288cf9a56f4e59afcd2aac86c
provenance:
  stars: 304568
  license: MIT
  last_push: 2026-10-27
  fetched: 2026-10-27
tags: [practice, testing]
---
Loads Matt Pocock's tdd before source under src/ changes while no test file has been edited this session, so the work goes red-green-refactor.
