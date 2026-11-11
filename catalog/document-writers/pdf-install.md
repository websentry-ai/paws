---
name: pdf-install
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?((python3?\s+-m\s+)?pip3?\s+install|uv\s+pip\s+install|uv\s+add|poetry\s+add)(\s+((-t|--target|--prefix|-r|--requirement|-c|--constraint|-i|--index-url|--extra-index-url|--root|--src|--group|--extra|--optional)(=[^\s;&|]+|\s+[^\s;&|]+)|-U|--upgrade|--user|-q|--quiet|--no-deps|--pre|--no-cache-dir|--force-reinstall|-I|--ignore-installed|--dev|-D|[^\s;&|-][^\s;&|]*))*\s+\"?(pypdf2?|PyPDF2|pdfplumber|reportlab|pypdfium2|pdf2image|pytesseract)\"?([=<>~!]\S*)?($|[\s;&|)])|(^|[;&|\n(])\s*(sudo\s+)?(npm\s+(i|install|add)|pnpm\s+(i|install|add)|yarn\s+(global\s+)?add|bun\s+(i|install|add))(\s+((--filter|-F|-w|--workspace|--prefix|-C|--dir|--cwd|--registry)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|-d|--dev|--exact|-W|--ignore-workspace-root-check|--legacy-peer-deps|--no-save|--ignore-scripts|[^\s;&|-][^\s;&|]*))*\s+(pdf-lib|pdfjs-dist)(@[^\s;&|)]*)?(\s|$|[;&|)])
action:
  type: require-skill
  skill: pdf
upstream:
  repo: github.com/anthropics/skills
  path: skills/pdf/SKILL.md
  ref: 2c5958ecac7cee2b583967a2c4981d0f4a75c579
  content_hash: sha256:48032992527f297b532782a85f9d4132417805eff1630944009bceb990d93f9a
provenance:
  stars: 192604
  license: NOASSERTION
  last_push: '2026-11-06'
  fetched: '2026-11-12'
tags:
- practice
- document-writers
---
Loads pdf before PDF command-line tools run or a PDF library is installed, so it uses the right library or tool for extract, merge, split and forms.
