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

- Write a check for each rule **the skill stresses** that a call can show: its
  "must", "never" and "always" rules, not every style preference. A block is
  graded fixed only when every check that applies passes, so ten checks on one
  rule make "fixed" almost never true: a test with ten git checks fell from 42%
  fixed to 5%. Two to five checks is usual, and five is the most. A skill with
  one such rule gets one.
- **List first, then pick.** Before writing any check, write down every firm
  rule of the skill that a call on this rule file can show. Then pick up to
  five, the ones the skill stresses most. A skill with three such rules gets
  three checks, not one.
- Every check must fail on a real call that breaks the skill and pass on one that
  follows it. If you can't name both calls, drop the check.
- **Stay inside the skill.** Each check traces to a sentence in the skill text,
  with its own numbers and its own exceptions. Don't add a limit, exemption or
  pattern the skill doesn't state, and don't tune a pattern to particular past
  commands: it has to hold for calls nobody has made yet. When two skills
  disagree, follow the one this rule loads.
- **Measure what the sentence measures.** A number applies to the part the
  skill names. If the skill caps the summary and its template puts more after
  the summary (`<type>(<scope>): <summary>  [<issue-id>]`), that extra part
  doesn't count. If it caps the subject line, everything on the line counts.
- **Cover every form the skill names.** When the skill lists several commands
  or forms for one rule ("before `x delete`, `x purge` or `x rm -r`, list what
  will go"), the check covers each form the rule file can block: one `match`
  that takes them all, or one case per form. Go through the rule file's match
  list. Each command it can block should fall under a check, unless the skill
  says nothing about it.
- Use the values the user's copy of the skill uses. A skill can leave
  placeholders like `{{COMMIT_TYPES}}` or `{{PROTECTED_BRANCHES}}`. Read the
  installed skill (or its `config.yaml`) for the filled values. If they're unset,
  use the skill's stated default, and say which values you assumed. If the
  skill gives no default, still check the part of the rule that doesn't need
  the value (that a step ran first, say), and leave out only the part that
  does.
- `label` says the rule in plain words, as a person would read it on a
  dashboard: `Ran tests, a build or a linter first`.
- A check looks at one call at a time, plus `blocked`. It can't say "the delete
  used the exact expression that was listed" or "the tests ran after the last
  edit". Leave such rules out.

## 2. Shape and `when`

```yaml
expect:
  - id: format                      # lower-case letters, digits, -
    label: 'Commit message is <type>(<scope>): <summary>'
    when: retry                     # before | retry | after | never
    if: >-
      call.text matches "..."
```

All `when` values look only at the blocked agent's turn. `before`, `after`
and `never` look at every call in it, of any tool, not only calls the rule
matches: a rule that blocks a shell command can still check a later MCP or
Write call.

| `when` | Passes when |
|---|---|
| `retry` | the retry (the next call the rule matches that goes through) matches |
| `before` | a call earlier in the turn than the retry matches, or the retry itself does. Calls before the block count. Calls sent in the same parallel batch as the blocked call or the retry don't |
| `after` | a call logged after the retry matches (not one in its own parallel batch) |
| `never` | no call after the block matches |

If no later call matches the rule, there's no retry: the block is graded
"other route" or "stopped", and `retry`, `before` and `after` checks fail.
**Choose `when` by where the compliant call goes:**

- **The compliant call still matches the rule** (a fixed commit message, a PR
  with a body): use `retry`.
- **Following the skill means a different call the rule doesn't match**
  (`rm -rf` → `trash`, `tar -xPf` → `tar -xzf … -C dir`): a `retry` check on that
  can never pass. Use `never` for the bad form ("never extracted with
  `-P`"), and limit it to blocks of that command with `match` on `blocked`
  (§4): any `tar` block, not only a `tar -P` block, because Claude can add the
  bad form on the retry.
- **The compliant next call matches the rule but is a different action** (a
  GitHub write to `main` is blocked, and Claude then calls `create_branch`): it
  becomes the retry. Give each action its own case (§4), so the branch check
  applies only to file writes.
- **"Do Y first":** use `before`. A guard on the same line counts
  (`git diff --staged && git commit`), even though the output wasn't read first.
- **A Write or Edit retried with changed content** (a real key replaced by a
  placeholder) is still the retry when the rule matches its path. It's graded
  "same retry" only when both the path and the content are unchanged.
- **The rule's call has nothing the skill says about it** (a filesystem MCP
  write under a delete-safety skill): leave `expect` out. A block where no check
  applies is graded "same retry" or "changed, unchecked", never "fixed".

## 3. What a check sees

`if` is an [expr](https://expr-lang.org) expression over `call` (the call
checked) and `blocked` (the call paws denied). Both have:

| Field | What it is |
|---|---|
| `tool` | `Bash`, `Write`, `Edit`, `Read`, `Glob`, `mcp__github__create_pull_request` … |
| `commands` | Bash only: each command the line may run, quotes removed, `sudo`/`env`/`timeout` dropped. `cd x && sudo git push -f` → `["cd x", "git push -f"]` |
| `text` | Bash: the whole line. Write: the content. Edit: the new text. MCP: the input's text values joined. Read/Glob/Grep/LS: the path or pattern |
| `input.<field>` | the raw tool input: `input.message`, `input.body`, `input.file_path`, `input.pattern` |
| `mcp.server`, `mcp.tool`, `mcp.class` | the parts of an MCP tool name |

**Use `commands` to ask "did it run X".** Text inside a heredoc, a commit
message or an `echo` is not a command, so
`any(call.commands, # matches "^git push\\b")` ignores a script that only prints
`git push`. A shell that runs text (`bash -c '...'`, a heredoc into `sh`) is
followed. Anchor with `^`, because each entry starts with the program.

A command counts whether or not it ran, because paws can't see which side of
`&&`/`||` ran. `npm test || true` counts as running the tests. To refuse a
masked exit code, add `&& !(call.text matches "\\|\\| *true\\b")`.

**Use `text` to read content**: a commit message or a PR body. Where the message
sits depends on how the commit is written:

| Form | Message is in |
|---|---|
| `git commit -m "msg"` | `text`, and the `git commit` entry of `commands` (quotes removed) |
| `git commit -m "$(cat <<'EOF' … EOF)"` | `text`, and the `git commit` entry of `commands` |
| `git commit -F - <<'EOF' … EOF` | `text` only. A heredoc into a command isn't part of that command |

`blocked` lets a check depend on what was blocked, or compare the two calls:

```yaml
    cases:
      - match: >-
          any(blocked.commands ?? [], # matches "^tar\\b.* -[a-zA-Z]*P")   # only tar -P blocks
        if: >-
          call.tool == "Bash" && any(call.commands, # matches "^tar\\b.* -[a-zA-Z]*P")
```

expr also has `let x = …;`, `in [...]`, `len`, `trim`, `lower`, `contains`,
`startsWith`, `endsWith`, `indexOf`, `lastIndexOf`, slicing `s[a:b]`, `split`,
and `??` for a missing field.

## 4. Several routes, or several actions: `cases`

The same rule can be followed through git, the gh CLI or an MCP tool, and one
rule file can match several commands (`git commit`, `git push`, `gh pr create`).
Give the check one case per route. The first case whose `match` holds supplies
the `if`. In a `match`, `call` is the retry (or the blocked call when there's no
retry) and `blocked` is the blocked call, for every `when`. To pick a case by
what was blocked, as `never` and `before` checks usually should, match on
`blocked`:

```yaml
  - id: format
    label: 'Commit message (or PR title) is <type>(<scope>): <summary>'
    when: retry
    cases:
      - match: >-
          call.tool == "Bash" && any(call.commands, # matches "^git( -[Cc] \\S+)* commit\\b")
          && call.text matches "commit\\b[^\\n]*( (-[a-zA-Z]*m|--message)|<<)"
        if: >-
          let m = "commit( +(-[a-ln-zA-Z]+|--[a-z][a-z-]*(=\\S+)?))* +(-[a-zA-Z]*m|--message)[ =]?[\"']";
          let h = "commit\\b[^\\n]*<<-? *[\"']?\\w+[\"']?[^\\n]*\\n";
          call.text matches ("(" + m + "|" + h + ")(feat|fix|chore|docs|refactor|test)(\\([^)\\n]*\\))?!?: \\S")
      - match: >-
          call.tool == "Bash" && any(call.commands, # matches "^gh pr create\\b")
        if: >-
          call.text matches "(--title|-t)[ =][\"']?(feat|fix|chore|docs|refactor|test)(\\([^)]*\\))?!?: \\S"
      - match: >-
          call.tool matches "__(create_or_update_file|push_files)$"
        if: >-
          (call.input.message ?? "") matches "^(feat|fix|chore|docs|refactor|test)(\\([^)]*\\))?!?: \\S"
```

When no case matches (a `git push` for a commit check), the check **doesn't
apply** and is left out of the grade. Never write `!isCommit || ...`, which
makes the check pass on every call it isn't about and inflates the pass rate.

- **Leave out calls that can't be graded.** A commit whose message isn't on the
  command line (`--no-edit`, `-F file`, `--fixup`) is filtered out by the first
  `match` above, which requires `-m` or a heredoc.
- **Scope `never` checks the same way.** "No push to main" on a block of a
  `git tag` passes trivially. Give it a case with `match` on `blocked` (a
  `git push`), so it counts only on blocks of the command it's about. Match the
  command, not the bad form: a block of a plain `git push` whose retry pushes to
  `main` must still count.

## 5. Patterns that work (tested)

| Need | Expression |
|---|---|
| A git subcommand, with `-C dir` or `-c k=v` before it | `any(call.commands, # matches "^git( -[Cc] \\S+)* commit\\b")` |
| The commit **subject**, not any line | the `m` and `h` prefixes in §4's first case, then the subject pattern |
| A command that isn't one thing | `any(call.commands, # matches "^find\\b" && !(# matches " -delete\\b"))` |
| An MCP field that may be missing | `(call.input.body ?? "") != ""`, `len(trim(call.input.body ?? "")) >= 20` |
| Any client's copy of an MCP tool | `call.tool matches "__create_pull_request$"` (`mcp__github__…`, `mcp__plugin_github_github__…`) |

A subject over 72 characters fails. Each quote style stops at its own quote, so
`don't` isn't cut short:

```yaml
        if: >-
          let f = "commit( +(-[a-ln-zA-Z]+|--[a-z][a-z-]*(=\\S+)?))* +(-[a-zA-Z]*m|--message)[ =]?";
          let h = "commit\\b[^\\n]*<<-? *[\"']?\\w+[\"']?[^\\n]*\\n";
          !(call.text matches (f + "\"[^\"\\n]{73,}") || call.text matches (f + "'[^'\\n]{73,}") || call.text matches (h + "[^\\n]{73,}"))
```

Whether a trailing ticket like `[WEB-123]` counts depends on what the skill
caps (§1). If the skill caps the whole subject line, use the check above. If it
caps the summary and its format puts the ticket after the summary, leave the
ticket out: pass when the subject isn't over 72, or is over only because of the
ticket:

```yaml
        if: >-
          let t = " *\\[[A-Z][A-Z0-9]*-[0-9]+\\]";
          !(call.text matches (f + "\"[^\"\\n]{73,}") || call.text matches (f + "'[^'\\n]{73,}") || call.text matches (h + "[^\\n]{73,}"))
          || call.text matches (f + "\"[^\"\\n]{1,72}" + t + "(\"|\\n)")
          || call.text matches (f + "'[^'\\n]{1,72}" + t + "('|\\n)")
          || call.text matches (h + "[^\\n]{1,72}" + t + "\\n")
```

(with `f` and `h` as above).

A test, build or lint run. Installs aren't verification (`npm ci`,
`npm install`, `npx playwright install`), and neither is watching a run
(`pgrep jest`):

```yaml
        if: >-
          call.tool == "Bash" && any(call.commands, # matches "^(\\S*/)?((npm|pnpm|yarn|bun) (run )?(test|build|lint|typecheck)\\b|(npx|bunx) (jest|vitest|tsc|eslint|playwright test|biome (check|lint|ci))|go (test|vet|build)|cargo (test|check|clippy)|((uv|poetry) run )?(python3? (-m )?)?pytest|((uv|poetry) run )?python3?( \\S+)* manage\\.py test|make (test|check|lint)|(jest|vitest|tsc|eslint|ruff|mypy|biome (check|lint))\\b)")
```

**Check the patterns against how people really run things.** A team may run
tests through `./node_modules/.bin/biome`, a Django `manage.py test`, or a
wrapper script. Search a few recent sessions for the command
(`grep -h '"command"' ~/.claude/projects/*/*.jsonl | grep -o 'manage.py test[^"]*' | head`),
and make sure the pattern catches what turns up.

## 6. Writing them so they don't silently break

- **Use folded blocks (`>-`) for `match` and `if`**, as above. Inside them YAML
  changes nothing: `'`, `"`, `#` and `: ` are literal, and lines join with
  spaces, so long expressions stay readable. With a single-quoted value instead,
  write `'` as `''`; unquoted, ` #` starts a comment.
- **expr strings.** Inside `"..."`, a regex backslash is `\\` (`\\s`, `\\b`), a
  `"` is `\"`, and `'` is plain. `\'` is an invalid escape, and the check is
  dropped.
- **Go regex (RE2).** There's no lookahead, lookbehind or backreference. Split
  the condition with `&&` and `!` instead.
- **`never` checks.** Guard with the tool first (`call.tool == "Bash" && ...`).
  An error on a call counts as a hit, because it can't show the call didn't
  happen.
- **Anchoring.** `^` belongs on `commands` entries, not on `text`. The text often
  starts with `cd ... &&`.
- **Over-matching.** `\\bjest\\b` anywhere in the text also matches
  `pgrep -f jest` and `ls jest.config.js`. Anchor on the command's start. A
  subject pattern that isn't tied to the subject passes on any body line.

## 7. Check your checks

```
unbound-paws explain 'git commit -m "fix: retry on 5xx"'
unbound-paws explain --tool mcp__github__create_pull_request '{"title":"x","body":"short"}'
unbound-paws explain --tool Write '{"file_path":"app/.env","content":"KEY=..."}'
unbound-paws explain 'cd web && npm test'
```

Any tool but Bash takes its whole input as JSON. Under each matching rule,
`explain` lists every check's result with that call as the retry. A call that a
rule doesn't match, but that one of its `before` or `after` checks counts (the
`npm test` above, for a commit rule), is listed under "would count toward":

```
  → git      event=pre-tool  action=require-skill  .paws/pre-tool/git.md
      checks, with this call as the retry:
        format          pass
        attribution     FAIL
        no-force        pass (this call doesn't break it)

this call would count toward these checks of other rules:
  verify-before-commit / verified (before)
```

In `explain`, `blocked` is the same call as `call`. To test a case that
reads `blocked`, put the blocked form in the command.

A check that doesn't compile appears as a `! problem in …` line naming the
check, and is dropped. Then try one
call of each kind the rule file blocks (each command, tool or route in its
match list). Where the skill has a firm rule for that kind of call, at least
one check must apply to it, not show "doesn't apply". For each
check, try at least one call that should pass and one that should fail,
including the tricky ones: a heredoc, `cd dir && ...`, a quoted message, an
apostrophe, the MCP route, and a commit with `--no-edit` (it should show
"doesn't apply").

## 8. Before you finish

Go back to the list you made in §1 for each rule file:

- Every firm rule on the list has a check, or a line in your notes saying why
  it has none (a call can't show it, or five checks are already taken by rules
  the skill stresses more).
- Every kind of call the rule file blocks has a check that applies to it, if
  the skill has a firm rule for it.
- No check fails a call that follows the skill. Try the compliant form of each
  check's rule once more, written the way an agent would (`cd dir && …`, a
  heredoc, the MCP route).
