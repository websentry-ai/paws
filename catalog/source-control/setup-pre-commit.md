---
name: setup-pre-commit
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?((npx\s+(-y\s+)?|bunx\s+|pnpm\s+(exec\s+|dlx\s+)?|yarn\s+(dlx\s+)?)husky\s+init\b|(npm(([ \t]|\\\n)+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force|[^\s;&|-][^\s;&|]*))*|pnpm(([ \t]|\\\n)+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts|[^\s;&|-][^\s;&|]*))*|yarn(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check))*([ \t]|\\\n)+(global([ \t]|\\\n)+)?add(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check|[^\s;&|-][^\s;&|]*))*|bun(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer|[^\s;&|-][^\s;&|]*))*)([ \t]|\\\n)+["']?(husky|lint-staged)(@[^\s;&|)"']*)?["']?([\s;&|)]|$))
action:
  type: require-skill
  skill: setup-pre-commit
upstream:
  repo: github.com/mattpocock/skills
  path: skills/misc/setup-pre-commit/SKILL.md
  ref: 4ae2317ddb22b046e26455e4f9c9b3aaaf62212d
  content_hash: sha256:98523a8db5977c41a095834df8697123bb845ff6ddbb9500733b161f11441a37
provenance:
  stars: 321219
  license: MIT
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Sets up Husky pre-commit hooks with lint-staged Prettier, typecheck and test scripts in a JS repo.
tags:
- practice
- source-control
---
Loads setup-pre-commit before husky or lint-staged is installed, so the hook, lint-staged config and Prettier setup match the recipe.
