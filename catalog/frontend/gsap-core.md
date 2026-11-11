---
name: gsap-core
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npm(([ \t]|\\\n)+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force|[^\s;&|-][^\s;&|]*))*|pnpm(([ \t]|\\\n)+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts|[^\s;&|-][^\s;&|]*))*|yarn(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check))*([ \t]|\\\n)+(global([ \t]|\\\n)+)?add(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check|[^\s;&|-][^\s;&|]*))*|bun(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer|[^\s;&|-][^\s;&|]*))*)([ \t]|\\\n)+["']?(gsap|@gsap/react)(@[^\s;&|)"']*)?["']?($|[\s;&|)])
action:
  type: require-skill
  skill: gsap-core
upstream:
  repo: github.com/greensock/gsap-skills
  path: skills/gsap-core/SKILL.md
  ref: aed9cfd3277740755f6bfc1155c7aa645403b760
  content_hash: sha256:dfc8ba849fa8b81c8cc60964ee2a5fa8db4b2a2611801cf549a5b267b8284fa5
provenance:
  stars: 18261
  license: MIT
  last_push: '2026-07-29'
  fetched: '2026-11-12'
summary: 'Writes GSAP core animations: tweens, easing, staggers, defaults and matchMedia for responsive and reduced motion.'
tags:
- practice
- frontend
---
Loads gsap-core before GSAP is installed, so tweens, easing and reduced-motion handling use the core API correctly.
