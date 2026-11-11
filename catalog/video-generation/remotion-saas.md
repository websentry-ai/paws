---
name: remotion-saas
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npm(([ \t]|\\\n)+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force|[^\s;&|-][^\s;&|]*))*|pnpm(([ \t]|\\\n)+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts|[^\s;&|-][^\s;&|]*))*|yarn(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check))*([ \t]|\\\n)+(global([ \t]|\\\n)+)?add(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check|[^\s;&|-][^\s;&|]*))*|bun(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer|[^\s;&|-][^\s;&|]*))*)([ \t]|\\\n)+["']?@remotion/(player|lambda|renderer|vercel|cloudrun|web-renderer)(@[^\s;&|)"']*)?["']?($|[\s;&|)])
action:
  type: require-skill
  skill: remotion-saas
upstream:
  repo: github.com/remotion-dev/skills
  path: skills/remotion-saas/SKILL.md
  ref: 1af7e69a2a613437d46095924057970a6a7df40d
  content_hash: sha256:a7bd14b6dc7a2fa6f2064d48abe884b4cbc934965bbce224c9b6f1f56e4f96c7
provenance:
  stars: 6813
  license: NOASSERTION
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: 'Guides building an app on Remotion: picking a template, embedding the Player, and choosing a render backend.'
tags:
- practice
- video-generation
---
Loads remotion-saas before the Remotion Player or a server render package is installed, so the template and render-backend choice is informed.
