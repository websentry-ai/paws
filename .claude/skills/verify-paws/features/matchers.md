# MCP and file matchers

## Sub-features

- `tool: mcp__server__*`: prefix wildcard, also inside an `A|B` list.
- `mcp_action: send|create|update|delete` (or a pipe list): classifies an MCP tool
  by the verbs in its own name, whatever the server prefix. Reads are never
  classified. Drafts and comments count as sends.
- `file_glob`: `**`, `*`, `?`, `{a,b}`; relative globs match at any directory
  boundary of the absolute path. Only Read, Write and Edit can match.

## How to get to it (user POV)

`matcher.mcp_action: send` for "any outgoing message"; `matcher.file_glob` for file types.

## Driving it with drive.py

- `run mcp-send-require-skill`: fixtures/fakechat_mcp.py serves `send_message`.
  Proof: one deny on `mcp__fakechat__send_message`, humanizer invoked, and the
  message written to `sent.jsonl` ends with the skill's `HUMAN-OK` token.
- `run file-glob-require-skill`: Write of `styles/app.css` is denied, the
  skill is invoked, and the file starts with `/* PROBE-CSS */`.

## Gotchas

- Case MCP servers come from `cases/<name>/mcp.json`, loaded with
  `--strict-mcp-config` so the user's own MCP servers stay out.
- The verb table lives in `internal/rule/mcp.go`; its test lists real tool
  names from Slack, Gmail, Notion, Linear, GitHub, Drive and Calendar.
