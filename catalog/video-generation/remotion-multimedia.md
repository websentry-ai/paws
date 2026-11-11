---
name: remotion-multimedia
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npm(([ \t]|\\\n)+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force|[^\s;&|-][^\s;&|]*))*|pnpm(([ \t]|\\\n)+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts|[^\s;&|-][^\s;&|]*))*|yarn(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check))*([ \t]|\\\n)+(global([ \t]|\\\n)+)?add(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check|[^\s;&|-][^\s;&|]*))*|bun(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer|[^\s;&|-][^\s;&|]*))*)([ \t]|\\\n)+["']?(@mediabunny/[\w.-]+|mediabunny)(@[^\s;&|)"']*)?["']?($|[\s;&|)])
action:
  type: require-skill
  skill: remotion-multimedia
upstream:
  repo: github.com/remotion-dev/skills
  path: skills/remotion-multimedia/SKILL.md
  ref: 1af7e69a2a613437d46095924057970a6a7df40d
  content_hash: sha256:8300b4afa850128722c141a9600fd2aa4a448f186e09b7ff9dbf5257545a4573
provenance:
  stars: 6813
  license: NOASSERTION
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Uses Mediabunny to read audio/video duration and dimensions and handle media in the browser.
tags:
- practice
- video-generation
---
Loads remotion-multimedia before Mediabunny is installed, so duration and dimension reads use its documented APIs.
