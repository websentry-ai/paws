---
name: "tdd-mattpocock"
event: "pre-tool"
matcher:
  tool: "Write|Edit"
  file_glob: "src/**/*.{ts,tsx,js,jsx,mjs,cjs,go,py,rs,rb,java,kt,swift,c,cc,cpp,h,hpp,cs,php}"
action:
  type: require-skill
  skill: "tdd"
---
Source under src/ changes while no test file has been edited this session
