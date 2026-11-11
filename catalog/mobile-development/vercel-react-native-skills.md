---
name: vercel-react-native-skills
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*((npx\s+(-y\s+)?|bunx\s+|pnpm\s+dlx\s+|yarn\s+dlx\s+)?create-expo-app|(npm|pnpm|yarn|bun)\s+create\s+expo(-app)?|(npx\s+|bunx\s+|pnpm\s+exec\s+)?expo\s+install)($|[^\w-])
action:
  type: require-skill
  skill: vercel-react-native-skills
upstream:
  repo: github.com/vercel-labs/agent-skills
  path: skills/react-native-skills/SKILL.md
  ref: 1a2c79d4e6c4a2283dc2c614b6538cbe1ed5ca62
  content_hash: sha256:14e7894ca92cbc7a23061df0085897c03e5e8d1f655c602dd1e9d7ac43e6faf0
provenance:
  stars: 33862
  license: NOASSERTION
  last_push: '2026-10-28'
  fetched: '2026-11-12'
summary: 'Applies React Native and Expo performance rules: list virtualization, GPU-friendly animation, native modules.'
tags:
- practice
- mobile-development
---
Loads vercel-react-native-skills before an Expo app is created or Expo packages are installed, so list, animation and native-module rules apply.
