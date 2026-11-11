---
name: stripe-best-practices
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*((npm(([ \t]|\\\n)+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force|[^\s;&|-][^\s;&|]*))*|pnpm(([ \t]|\\\n)+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts|[^\s;&|-][^\s;&|]*))*|yarn(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check))*([ \t]|\\\n)+(global([ \t]|\\\n)+)?add(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check|[^\s;&|-][^\s;&|]*))*|bun(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer|[^\s;&|-][^\s;&|]*))*)(([ \t]|\\\n)+["']?stripe(@[^\s;&|)"']*)?|([ \t]|\\\n)+["']?@stripe/(stripe-js|react-stripe-js|stripe-react-native|connect-js|react-connect-js)(@[^\s;&|)"']*)?)["']?($|[\s;&|)])|(python3?\s+-m\s+pip\s+install|pip3?\s+install|uv\s+pip\s+install|uv\s+add|poetry\s+add)(([ \t]|\\\n)+((-t|--target|--prefix|-r|--requirement|-c|--constraint|-i|--index-url|--extra-index-url|--root|--src|--group|--extra|--optional)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|-U|--upgrade|--user|-q|--quiet|--no-deps|--pre|--no-cache-dir|--force-reinstall|-I|--ignore-installed|--dev|-D|[^\s;&|-][^\s;&|]*))*([ \t]|\\\n)+["']?stripe((==|>=|~=)\S+)?["']?($|[\s;&|)])|gem\s+install\s+stripe["']?($|[\s;&|)])|bundle\s+add\s+stripe["']?($|[\s;&|)])|composer\s+require\s+stripe/stripe-php|go\s+get\s+github\.com/stripe/stripe-go|dotnet\s+add\s+package\s+Stripe\.net)
action:
  type: require-skill
  skill: stripe-best-practices
upstream:
  repo: github.com/stripe/ai
  path: skills/stripe-best-practices/SKILL.md
  ref: 8570d472fbc4ff2c84329af1d15030ecfdea3570
  content_hash: sha256:61423cb1692bd8bbb3c3b3db46fc6ff65f4e24e0246ef8d7963371bd28ecad63
provenance:
  stars: 2054
  license: MIT
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: 'Guides Stripe integration choices: Checkout vs PaymentIntents, Connect, billing, tax, sandboxes, key security.'
tags:
- practice
- payments
---
Loads stripe-best-practices before a Stripe SDK is installed, so the current API version and integration choices are used.
