---
name: notion-mcp
event: pre-tool
matcher:
  tool: mcp__notion__notion-create-pages|mcp__notion__notion-update-page|mcp__notion__notion-move-pages|mcp__notion__notion-duplicate-page|mcp__notion__notion-create-database|mcp__notion__notion-update-data-source|mcp__notion__notion-create-comment|mcp__claude_ai_Notion__notion-create-pages|mcp__claude_ai_Notion__notion-update-page|mcp__claude_ai_Notion__notion-move-pages|mcp__claude_ai_Notion__notion-duplicate-page|mcp__claude_ai_Notion__notion-create-database|mcp__claude_ai_Notion__notion-update-data-source|mcp__claude_ai_Notion__notion-create-comment
action:
  type: require-skill
  skill: notion
  on_miss: digest
  digest: digests/notion.md
tags: [org, docs]
---
Loads the notion skill before Notion pages, databases or comments are created, edited, moved or archived.
