---
name: setup-pre-commit
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?((npx\s+(-y\s+)?|bunx\s+|pnpm\s+(exec\s+|dlx\s+)?|yarn\s+(dlx\s+)?)husky\s+init\b|(npm|pnpm|yarn|bun)\s+(i|install|add)\b[^;&|\n]*\s(husky|lint-staged)(@\S+)?([\s;&|)]|$))
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
