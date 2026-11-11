---
name: pdf-install
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?((python3?\s+-m\s+)?pip3?\s+install|uv\s+pip\s+install|uv\s+add|poetry\s+add)([ \t]+((-t|--target|--prefix|-r|--requirement|-c|--constraint|-i|--index-url|--extra-index-url|--root|--src|--group|--extra|--optional)(=[^\s;&|]+|[ \t]+[^\s;&|]+)|-U|--upgrade|--user|-q|--quiet|--no-deps|--pre|--no-cache-dir|--force-reinstall|-I|--ignore-installed|--dev|-D|[^\s;&|-][^\s;&|]*))*[ \t]+["']?(pypdf2?|PyPDF2|pdfplumber|reportlab|pypdfium2|pdf2image|pytesseract)\"?([=<>~!]\S*)?["']?($|[\s;&|)])|(^|[;&|\n(])\s*(sudo\s+)?(npm\s+(i|install|add)([ \t]+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|[ \t]+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force|[^\s;&|-][^\s;&|]*))*|pnpm\s+(i|install|add)([ \t]+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|[ \t]+[^\s;&|]+)|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts|[^\s;&|-][^\s;&|]*))*|yarn\s+(global\s+)?add([ \t]+((--cwd)(=[^\s;&|]+|[ \t]+[^\s;&|]+)|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check|[^\s;&|-][^\s;&|]*))*|bun\s+(i|install|add)([ \t]+((--cwd)(=[^\s;&|]+|[ \t]+[^\s;&|]+)|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer|[^\s;&|-][^\s;&|]*))*)[ \t]+["']?(pdf-lib|pdfjs-dist)(@[^\s;&|)]*)?["']?(\s|$|[;&|)])
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
