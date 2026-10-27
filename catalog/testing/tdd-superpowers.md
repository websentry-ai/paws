---
name: tdd-superpowers
event: pre-tool
matcher:
  tool: Write|Edit
  file_glob: 'src/**/*.{ts,tsx,js,jsx,mjs,cjs,go,py,rs,rb,java,kt,swift,c,cc,cpp,h,hpp,cs,php}'
when: 'no-edit-matching:{*_test.*,*.test.*,*.spec.*,test_*,*_spec.*,__tests__/**,tests/**,test/**,spec/**}'
action:
  type: require-skill
  skill: test-driven-development
upstream:
  repo: github.com/obra/superpowers
  path: skills/test-driven-development/SKILL.md
  ref: 2ce8aa15d05bd57213e6f746f953aafb7ded7a8b
  content_hash: sha256:2782702b2cf69ace085ed30d067b6f8bfd2d777ad51770fc5bdae9cb5c1b5c04
provenance:
  stars: 305962
  license: MIT
  last_push: 2026-10-23
  fetched: 2026-10-27
tags: [practice, testing]
---
Loads superpowers' test-driven-development before source under src/ changes while no test file has been edited this session, so the failing test comes first.
