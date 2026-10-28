---
name: paws
description: Write, change or debug paws rules (.paws/*.md) that gate Claude Code tool calls. Use when the user asks to always load a skill before an action ("always use humanizer when I write PR bodies"), to block, warn about or audit a command, file type or MCP send, or asks why a paws gate fired.
---

# paws rules

paws is a hook that reads `.paws/**/*.md` (this repo) and the user dir
(`$PAWS_HOME` when set, else `~/.paws/`) on every tool call. Each file is one rule. A rule takes effect on the
next tool call; nothing needs restarting.

## Turn the request into a rule

1. **What action?** Map it to an event and matcher (tables below).
2. **What should happen?** Usually `require-skill`: deny until the skill is
   loaded, then allow. `advise` for a gentle note, `block` for never,
   `warn` to ask the human, `audit` to only record.
3. **Where?** `.paws/<event>/<name>.md` in the repo for the team; the user
   dir for this user in every repo ("all my repos", "everywhere"). The user
   dir is the last path on the `.paws :` line of any `paws explain` output;
   it is `$PAWS_HOME` when that is set, so never assume `~/.paws`. Ask if
   unclear.
4. **Check it** with `paws explain` (below) before you report it done.

Run `paws` as a plain command, one `paws explain` per Bash call: no `||`,
`;`, `&&`, loops or variables around it, which need approval. Only if
`paws` is not found, run `~/.paws/bin/paws` (or `$PAWS_HOME/bin/paws`).

Write the file with frontmatter, then one or two sentences saying why:

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
PR titles and bodies read like a person wrote them.
```

`name` is unique, lowercase, dashes. `skill` is the skill's `name:` exactly
as Claude Code lists it.

## Common requests

| Request | event | matcher | action |
|---|---|---|---|
| skill X before PR bodies | `pre-tool` | `tool: Bash`, `command_regex: (^\|[;&\|\n])\s*gh\s+pr\s+(create\|edit)\b` | `require-skill` |
| skill X before commits | `pre-commit` | `tool: Bash` | `require-skill` |
| skill X before pushes | `pre-push` | `tool: Bash` | `require-skill` |
| skill X before any Slack/Gmail/Notion/Linear send | `pre-tool` | `mcp_action: send` | `require-skill` |
| skill X when editing React files | `pre-tool` | `tool: Write\|Edit`, `file_glob: '**/*.{tsx,jsx}'` | `require-skill` |
| skill X at the start of every session | `session-start` | none | `require-skill` |
| never `rm -rf` (also `-fr`, `-Rf`, `sudo`) | `pre-tool` | `tool: Bash`, `command_regex: (^\|[;&\|\n])\s*(sudo\s+)?rm\s+(-\S+\s+)*-[a-zA-Z]*([rR][a-zA-Z]*f\|f[a-zA-Z]*[rR])` | `block` |
| ask me before `terraform apply` | `pre-tool` | `tool: Bash`, `command_regex: (^\|[;&\|\n])\s*terraform\s+apply\b` | `warn` |
| skill X only in repos using stripe | add `when: dependency:stripe` | | |

(`\|` in the table is a plain `|` in the file.) Split flags such as
`rm -r -f` need an `exec-script` check; a regex over the command line is not
a shell parser.

## Events

`pre-tool` (any tool call), `pre-commit` and `pre-push` (Bash running
`git commit` / `git push`), `user-prompt` (the prompt text),
`session-start` (`command_regex` sees `startup`, `resume`, `clear` or `compact`).

## Matcher (every field set must match)

- `tool`: `Bash`, `Read`, `Write`, `Edit`, `NotebookEdit`, `Grep`, `Glob`,
  `WebFetch`, `WebSearch`, `Agent`, an MCP name, `mcp__server__*`, or `A|B`.
- `command_regex`: Go RE2 over the target (Bash command, file path, URL,
  prompt, MCP input JSON). No lookahead or lookbehind. Start Bash patterns
  with `(^|[;&|\n])\s*` so chained commands match and echoed text does not.
  Leave regexes unquoted (a leading `(` or `^` is fine), or wrap them in
  single quotes when they contain `: ` or start with `*`, `&`, `!` or `{`.
  Never use double quotes: YAML reads `\s` and `\b` inside them as escapes,
  so the rule fails to parse or silently never matches.
- `file_glob`: `**`, `*`, `?`, `{a,b}`, for Read/Write/Edit/NotebookEdit.
  It matches Read too: add `tool: Write|Edit` to gate only changes. Quote a
  glob that starts with `*` (single quotes).
- `mcp_action`: `send`, `create`, `update`, `delete`.

## when (optional; all must hold)

`dependency:PKG`, `repo-marker:PATH`, `new-file`,
`no-edit-matching:GLOB` (e.g. no test edited yet), `agent-count>N`.

## Actions

| type | effect | needs |
|---|---|---|
| `require-skill` | deny until `skill` is loaded (last 20 turns), then allow | `skill` |
| `advise` | allow and show `message` once per window | `message` |
| `block` | deny, no retry | optional `deny_message` |
| `warn` | ask the human | optional `deny_message` |
| `audit` | allow, record only | — |
| `prompt-inject` | add `message` to the prompt (`user-prompt` only) | `message` |
| `exec-script` | run `script` beside the rule; exit 2 denies | `script` |

## Check it

```
paws explain 'gh pr create --fill'            # Bash command
paws explain --tool Edit src/App.tsx          # file tool
paws explain --event session-start startup
```

The output marks matching rules with `→` and prints the decision. If a rule
does not show up, the file failed to parse: `paws explain` lists it with the
reason. Check one command that should match and one near-miss that should
not. A `block` rule also fires on your own Bash call when the quoted test
command is chained (`'cd x && rm -rf y'`): that deny shows the rule matches.
Then tell the user what the rule does and where it lives.

## When a gate fires on you

- "This action requires the X skill. Invoke it with the Skill tool...":
  invoke X with the Skill tool, then retry the same command. Do not rewrite
  the command to dodge the rule.
- A deny that starts with `paws digest (...)` carries the skill's rules: no
  Skill call is needed. Apply the rules, re-forming the command if they say
  so, and retry.
- "Blocked by paws rule ...": stop and tell the user. Do not look for another
  way to do the same thing.
- If X cannot be loaded (no such skill), or the same deny comes back after you
  loaded it, stop and tell the user which rule and skill are involved. Do not
  keep retrying.

## Packs

Ready-made rules: `paws search <text>`, `paws add <pack>` (`--project` to
install into the repo), `paws list`, `paws remove <pack>`.
