---
name: analytics
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?(npm(([ \t]|\\\n)+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force|[^\s;&|-][^\s;&|]*))*|pnpm(([ \t]|\\\n)+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts|[^\s;&|-][^\s;&|]*))*|yarn(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check))*([ \t]|\\\n)+(global([ \t]|\\\n)+)?add(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check|[^\s;&|-][^\s;&|]*))*|bun(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer))*([ \t]|\\\n)+(i|install|add)(([ \t]|\\\n)+((--cwd)(=[^\s;&|]+|([ \t]|\\\n)+[^\s;&|]+)|--no-[\w-]+|--[\w-]+=[^\s;&|]+|--silent|-s|--quiet|--verbose|-v|--prefer-offline|--prefer-online|--offline|--frozen-lockfile|--package-lock-only|--foreground-scripts|--dry-run|--break-system-packages|--disable-pip-version-check|--require-hashes|--isolated|--compile|--prefer-binary|--ignore-engines|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer|[^\s;&|-][^\s;&|]*))*)([ \t]|\\\n)+["']?(posthog-js|posthog-node|mixpanel|mixpanel-browser|@segment/analytics-next|@segment/analytics-node|react-ga4|@amplitude/analytics-browser|@amplitude/analytics-node)(@[^\s;&|)]*)?["']?(\s|$|[;&|)])
action:
  type: require-skill
  skill: analytics
upstream:
  repo: github.com/coreyhaines31/marketingskills
  path: skills/analytics/SKILL.md
  ref: 3e729622e3de0c82abdb906190d91f9018045872
  content_hash: sha256:064ca10386885a91dc0a7251fe6993606df4e37344d620370d8744a26c970938
provenance:
  stars: 59621
  license: MIT
  last_push: '2026-11-11'
  fetched: '2026-11-12'
summary: 'Plans and audits analytics tracking: event naming, tracking plans, GA4, GTM, Mixpanel, PostHog, UTMs.'
tags:
- practice
- data-analytics
---
Loads analytics before an analytics SDK is installed, so events are named and planned before tracking code is added.
