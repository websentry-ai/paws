# Writing expect checks

An `expect` block on a rule says how to tell, after paws blocked a call and Claude
loaded the skill, whether Claude then did what the skill asks. paws grades the
checks at the end of the turn, on the turn's own tool calls, and keeps only pass
or fail. Checks never change what the rule blocks.

## 1. Pick what to check

Read the skill and list its rules that **show in a tool call**: a commit message
format, a flag that must or mustn't appear, a command that must run first, a
field an MCP call must fill. Skip rules about thinking, wording to the user, or
anything no call carries.

- Write one check per rule, and two to four per rule file. A check nobody can
  fail measures nothing.
- Every check should fail on a real call that breaks the skill and pass on one
  that follows it. If you can't name both calls, drop the check.
- `label` says the rule in plain words, as a person would read it on a
  dashboard: `Ran tests, a build or a linter first`.

## 2. Shape

```yaml
expect:
  - id: format                      # lower-case letters, digits, -
    label: 'Commit message is <type>(<scope>): <summary>'
    when: retry                     # before | retry | after | never
    if: '...'                       # or cases:, below
```

`when`, all within the blocked agent's turn:

| `when` | Passes when |
|---|---|
| `retry` | the retry (the next call the rule matches that goes through) matches |
| `before` | a call earlier in the turn than the retry, the retry included, matches. Calls before the block count; the blocked call's own parallel batch doesn't |
| `after` | a call after the retry matches |
| `never` | no call after the block matches |

Use `retry` for "the call itself must look like X", `before` for "do Y first",
and `never` for "don't ever do Z".

## 3. What a check sees

`if` is an [expr](https://expr-lang.org) expression over `call` (the call
checked) and `blocked` (the call paws denied):

| Field | What it is |
|---|---|
| `call.tool` | `Bash`, `Write`, `Edit`, `Glob`, `mcp__github__create_pull_request` … |
| `call.commands` | Bash only: each command the line runs, quotes removed, `sudo`/`env`/`timeout` dropped. `cd x && sudo git push -f` → `["cd x", "git push -f"]` |
| `call.text` | the whole Bash line, what a Write writes, an Edit's new text, an MCP call's text values joined |
| `call.input.<field>` | the raw tool input: `call.input.message`, `call.input.body` |
| `call.mcp.server`, `.tool`, `.class` | the parts of an MCP tool name |

**Use `commands` to ask "did it run X".** Text inside a heredoc, a commit
message, or an `echo` is not a command, so `any(call.commands, # matches "^git push\\b")`
ignores a Python script that only prints `git push`. A shell that runs text
(`bash -c '...'`, a heredoc into `sh`) is followed. Anchor with `^`: each entry
starts with the program.

**Use `text` to read content**, such as a commit message (which may be in a
heredoc: `git commit -F - <<'EOF'`) or a PR body.

## 4. One skill, several routes: `cases`

The same rule can be followed through git, the gh CLI or an MCP tool, and one
rule file can match several commands (`git commit`, `git push`, `gh pr create`).
Give the check one case per route. The first case whose `match` holds on the
retry (or on the blocked call when there's no retry) supplies the `if`:

```yaml
  - id: format
    label: 'Commit message (or PR title) is <type>(<scope>): <summary>'
    when: retry
    cases:
      - match: 'call.tool == "Bash" && any(call.commands, # matches "^git( -C \\S+)* commit\\b")'
        if: 'call.text matches "\\b(feat|fix|chore|docs|refactor|test)(\\([^)]*\\))?!?: \\S"'
      - match: 'call.tool == "Bash" && any(call.commands, # matches "^gh pr create\\b")'
        if: 'any(call.commands, # matches "^gh pr create\\b.* (--title|-t)[ =][\"'']?(feat|fix|chore|docs|refactor|test)(\\([^)]*\\))?!?: \\S")'
      - match: 'call.tool matches "__(create_or_update_file|push_files)$"'
        if: '(call.input.message ?? "") matches "^(feat|fix|chore|docs|refactor|test)(\\([^)]*\\))?!?: \\S"'
```

When no case matches (a `git push` for a commit check), the check **doesn't
apply** and is left out of the grade. Never write `!isCommit || ...`, which
makes the check pass on every call it isn't about and inflates the pass rate.
Use cases instead. Leave a call out with `match` when it can't be graded, too:
a commit whose message isn't on the command line (`--no-edit`, `-F file`,
`--fixup`) can't be checked for format.

## 5. Patterns that work

| Need | Expression |
|---|---|
| A git subcommand, with `-C dir` or `-c k=v` before it | `any(call.commands, # matches "^git( -[Cc] \\S+)* commit\\b")` |
| A command that isn't one thing | `any(call.commands, # matches "^find\\b" && !(# matches " -delete\\b"))` |
| A test runner, with a path or wrapper in front | `any(call.commands, # matches "^(\\S*/)?(npm (run )?test|npx (jest\|vitest)\|go test\|(python3? -m )?pytest\|cargo test)\\b")` |
| An MCP field that may be missing | `(call.input.body ?? "") != ""`, `len(trim(call.input.body ?? "")) >= 20` |
| Any of several MCP servers' tool | `call.tool matches "__(create_pull_request)$"` (works for `mcp__github__…` and `mcp__plugin_github_github__…`) |
| A length limit on part of a line | Regex can't measure a captured group. Bound each alternative: `feat[^\\n"'']{2,46}\|fix[^\\n"'']{2,47}` for a 50-character subject |

## 6. Mistakes that silently break a check

- **YAML.** Put every `match` and `if` in single quotes. Unquoted, ` #` starts a
  comment and `: ` starts a map. Inside single quotes, write `'` as `''`.
- **expr strings.** Inside `"..."`, a regex backslash is `\\` (`\\s`, `\\b`), a
  `"` is `\"`, and a `'` is plain `'` (YAML-doubled to `''`). `\'` is an invalid
  escape, and the check is dropped.
- **Go regex (RE2).** There's no lookahead, lookbehind or backreference. Split
  the condition with `&&` and `!` instead.
- **`never` checks.** Guard with the tool first (`call.tool == "Bash" && ...`).
  An error on a call counts as a hit, because it can't show the call didn't
  happen.
- **Anchoring.** `^` belongs on `call.commands` entries, not on `call.text`. The
  text often starts with `cd ... &&`.
- **Over-matching.** `\\bjest\\b` anywhere in the text also matches
  `pgrep -f jest` and `ls jest.config.js`. Anchor on the command's start.
- **Ticket tags.** Many orgs append `[ABC-123]`. Don't count it against a
  length limit another skill imposes.

## 7. Check your checks

```
unbound-paws explain 'git commit -m "fix: retry on 5xx"'
unbound-paws explain 'gh pr create --title "Add x" --body y'
unbound-paws explain --tool mcp__github__create_pull_request '{"title":"x","body":"short"}'
```

Under each matching rule, `explain` lists every check's result with that call
as the retry:

```
  → git      event=pre-tool  action=require-skill  .paws/pre-tool/git.md
      checks, with this call as the retry:
        format          pass
        attribution     FAIL
        no-force        pass (this call doesn't break it)
```

A check that doesn't compile appears as a `!` line and is dropped. For each
check, run at least one call that should pass and one that should fail,
including the tricky ones: a heredoc, `cd dir && ...`, a quoted message, the MCP
route. A `before` check shows whether this call would count; the rest of the
turn isn't known to `explain`.
