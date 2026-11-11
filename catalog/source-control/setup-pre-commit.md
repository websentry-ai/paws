---
name: setup-pre-commit
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?((npx\s+(-y\s+)?|bunx\s+|pnpm\s+(exec\s+|dlx\s+)?|yarn\s+(dlx\s+)?)husky\s+init\b|(npm\s+(i|install|add)(\s+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force|[^\s;&|-][^\s;&|]*))*|pnpm\s+(i|install|add)(\s+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts|[^\s;&|-][^\s;&|]*))*|yarn\s+(global\s+)?add(\s+((--cwd)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check|[^\s;&|-][^\s;&|]*))*|bun\s+(i|install|add)(\s+((--cwd)(=[^\s;&|]+|\s+[^\s;&|]+)|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer|[^\s;&|-][^\s;&|]*))*)\s+(husky|lint-staged)(@\S+)?([\s;&|)]|$))
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
