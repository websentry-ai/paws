---
name: paws
description: Any request that mentions paws, or a paws rule, is a rule to write in .paws/*.md, even when worded as an instruction to you ("load X before you edit Y, via paws"). Use to write, change or debug paws rules, the hook that gates Claude Code tool calls; when the user wants a skill loaded before an action, a command, file or MCP call blocked, approved, logged or annotated; or asks why a paws gate fired.
---

# paws rules

A rule is one markdown file: YAML frontmatter, then a line saying why. paws
reads every rule on every tool call, so a saved rule is live.

## Steps

1. **Read the rules that exist** in `.paws/` and the user dir. When one covers
   the same action, edit it instead of adding a second.
2. **Pin the request to concrete commands, files or MCP tools.** For a vague
   request, pick the narrowest reading and name it in your reply. paws only
   allows, denies, asks, logs or adds context; it never edits a command or
   its output. When asked for that, say so and offer the nearest rule.
3. **Write one file per rule.** Repo: `.paws/<event>/<name>.md`. "All my
   repos": the user dir, the last path on the `.paws :` line of
   `paws explain` (`$PAWS_HOME` when set, else `~/.paws`).
4. **Check it** with `paws explain` on one command that must match and one
   near-miss. Done when the first shows `→` on the rule and the expected
   decision, the second shows neither, and no `! skipped` line appears.
5. **Report** the path and what the rule does.

Run each `paws explain` as its own plain Bash call, nothing chained. If `paws`
isn't on PATH, use `$PAWS_HOME/bin/paws` or `~/.paws/bin/paws`. If the new
rule denies your own check, that deny proves the match; check the near-miss.

```markdown
---
name: humanizer-pr
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*gh\s+pr\s+(create|edit)\b
action:
  type: require-skill
  skill: humanizer
---
PR bodies read like a person wrote them.
```

`name` is unique, lowercase, dashes. `skill` is the skill's `name:` exactly.

## Events

- `pre-tool`: any tool call.
- `pre-commit`, `pre-push`: Bash running `git commit` or `git push`, chained or with
  flags like `-C dir`. The event does the matching; add no `command_regex`.
- `user-prompt`: the prompt text. No `warn` or `advise`.
- `session-start`: `command_regex` sees `startup`, `resume`, `clear` or
  `compact`; no other matcher. Only `require-skill`, `advise`, `audit`.

## Matcher (every field set must match)

- `tool`: `Bash`, `Read`, `Write`, `Edit`, `Agent`, an MCP tool, `A|B`, or a
  server wildcard `mcp__slack__*`. Omit for any tool.
- `command_regex`: Go RE2 over the Bash command, file path, URL, prompt or MCP
  input JSON. Start Bash patterns with `(^|[;&|\n])\s*` so chained commands
  match and quoted or echoed text doesn't; add `(sudo\s+)?` or
  `(\w+=\S*\s+)*` for prefixed forms. RE2 has no lookahead, so list the forms
  you want: `--force(\s|$)` spares `--force-with-lease`.
- `file_glob`: `**`, `*`, `?`, `{a,b}`. A relative glob matches at any
  directory (`*.py` is every Python file). It matches Read too; add
  `tool: Write|Edit` to gate changes only.
- `mcp_action`: `send`, `create`, `update`, `delete` or `A|B`, from the MCP
  tool's verb (`send_message` is send).

Write regexes and globs bare, or in single quotes when they start with `*`,
`{`, `&` or `!` or contain `: `. Double quotes make `\s` and `\b` YAML
escapes and break the rule.

## when (a string or a list; all must hold)

- `dependency:PKG`: in package.json, pyproject.toml, requirements.txt, go.mod or Cargo.toml.
- `repo-marker:PATH`: the path exists at the repo root.
- `new-file`: the target file doesn't exist yet.
- `no-edit-matching:GLOB`: no successful Edit or Write of a matching path this session.
- `agent-count>N`: more than N Agent calls since the user's last prompt.

## Actions

- `require-skill` + `skill`: deny until the skill was invoked in the last 20
  turns. With `on_miss: digest` and `digest: <path from the rule, under 7000
  bytes>`, the deny carries the digest text instead.
- `advise` + `message`: allow and add the message to context once per 20 turns.
- `block`: deny, no retry; the text names the rule, then `deny_message`.
- `warn`: ask the human; optional `deny_message`.
- `audit`: allow and log; the agent sees nothing.
- `prompt-inject` + `message`: add the message to the prompt (`user-prompt` only).
- `exec-script` + `script` (a path beside the rule): `/bin/sh` runs it with
  JSON `{event, tool, target, raw}` on stdin; exit 2 denies with stderr as the
  reason; 5 s limit.

## Check it

```
paws explain 'git push --force origin main'
paws explain --tool Edit src/App.tsx
paws explain --tool mcp__slack__send_message '{}'
paws explain --event user-prompt 'deploy to prod'
paws explain --event session-start startup
```

`audit` shows `→` with `allow (no matching gate)`. `exec-script` shows `→` but
never runs; run the script yourself with
`echo '{"target":"rm -rf x"}' | sh <script>` and read the exit code.
`paws explain` has no transcript: `agent-count` never holds there and
`no-edit-matching` always does.

## When a gate fires on you

"Requires the X skill": invoke X, retry the same command. A `paws digest`
deny carries the rules: apply them, retry. "Blocked by paws rule": stop and
tell the user. A repeat deny after loading X: stop and name the rule.

Ready-made rules: `paws search`, `paws add <pack>` (`--project` for this
repo), `paws list`, `paws remove`.
