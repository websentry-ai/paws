---
name: paws
description: Any request that mentions paws, or a paws rule, is a rule to write in .paws/*.md, even when worded as an instruction to you ("load X before you edit Y, via paws"). Use to write, change or debug paws rules, the hook that gates Claude Code tool calls; when the user wants a skill loaded before a command, file edit, MCP call or session start; or asks why a paws gate fired.
---

# paws rules

A rule is one markdown file: YAML frontmatter, then a line saying why. paws
reads every rule on every tool call, so a saved rule is live.

## Steps

1. **Read the rules that exist** in `.paws/` and the user dir. When one covers
   the same action, edit it instead of adding a second.
2. **Pin the request to concrete commands, files or MCP tools.** For a vague
   request, pick the narrowest reading and name it in your reply. paws only
   makes a skill load before a call. When asked for anything else (stopping
   a call for good, asking the user, recording calls, editing a command or
   its output), say so and offer a rule that loads a skill carrying the
   instruction.
3. **Write one file per rule.** Repo: `.paws/<event>/<name>.md`. "All my
   repos": the user dir, the last path on the `.paws :` line of
   `paws explain` (`$PAWS_HOME` when set, else `~/.paws`).
4. **Check it** with `paws explain` on one command that must match and one
   near-miss; for a Bash rule, make the near-miss that command inside
   `echo '...'`. Done when the first shows `→` on the rule and `DENY` naming
   the skill, the second shows neither, and no `! skipped` line appears.
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
- `session-start`: `command_regex` sees `startup`, `resume`, `clear` or
  `compact`; no other matcher. Can't deny: adds context naming the skill.

## Matcher (every field set must match)

- `tool`: `Bash`, `Read`, `Write`, `Edit`, `Agent`, an MCP tool, `A|B`, or a
  server wildcard `mcp__slack__*`. Omit for any tool.
- `command_regex`: Go RE2 over the Bash command, file path, URL or MCP input
  JSON. Start Bash patterns with `(^|[;&|\n])\s*` so chained commands
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

## Action

`require-skill` + `skill` is the only action: deny until the skill was
invoked in the last 20 turns. Optional `deny_message` replaces the deny text.
With `on_miss: digest` and `digest: <path from the rule, under 7000 bytes>`,
the deny carries the digest text instead. A rule with any other action is
skipped.

## Check it

```
paws explain 'git push --force origin main'
paws explain --tool Edit src/App.tsx
paws explain --tool mcp__slack__send_message '{}'
paws explain --event session-start startup
```

`paws explain` has no transcript: a skill never counts as loaded,
`agent-count` never holds and `no-edit-matching` always does.

## When a gate fires on you

"Requires the X skill": invoke X, retry the same command. A `paws digest`
deny carries the rules: apply them, retry. A repeat deny after loading X:
stop and name the rule.

Ready-made rules: `paws search`, `paws add <pack>` (`--project` for this
repo), `paws list`, `paws remove`.
