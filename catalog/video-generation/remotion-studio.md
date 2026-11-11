---
name: remotion-studio
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?([A-Z_][A-Z0-9_]*=\S*\s+)*(npx\s+(-y\s+|--yes\s+)?|bunx\s+(--bun\s+)?|pnpm\s+(dlx\s+|exec\s+)?|yarn\s+(dlx\s+)?|npm\s+exec\s+)?remotion(@\S*)?\s+(studio|preview)($|[^\w-])
action:
  type: require-skill
  skill: remotion-studio
upstream:
  repo: github.com/remotion-dev/skills
  path: skills/remotion-studio/SKILL.md
  ref: 1af7e69a2a613437d46095924057970a6a7df40d
  content_hash: sha256:6f0764115d1fb7fde62f97a819c4dbe7a5e04722a50e90f1a1a9dc86c53876a6
provenance:
  stars: 6813
  license: NOASSERTION
  last_push: '2026-11-09'
  fetched: '2026-11-12'
summary: Starts Remotion Studio early, opens the printed preview URL, and keeps it open while editing the video.
tags:
- practice
- video-generation
---
Loads remotion-studio before Remotion Studio is started, so the preview URL is opened and kept open while editing.
