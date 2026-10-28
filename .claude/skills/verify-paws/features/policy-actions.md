# block, warn, audit

## Sub-features

- `block`: hard deny with `deny_message` (or a default naming the rule).
- `warn`: Claude Code asks the human to approve.
- `audit`: allow silently and append a record to `$PAWS_HOME/audit.log`.
- Most restrictive verdict wins when several rules match.

## How to get to it (user POV)

A rule in `.paws/` with `action.type` set to `block`, `warn` or `audit`.

## Driving it with drive.py

- `run block-tag`: the agent's `git tag` is denied with the rule's message,
  and no tag exists afterwards.
- `run audit-status`: `git status` runs, the audit log has an `allow` record
  for the rule, and no tool result carries a paws message.

## Driving it with interactive.py

`warn` needs a human, so it runs in the real TUI (`scripts/interactive.py`):

- `run warn-approve`: the approval prompt shows the rule's `WRN-4F` message;
  answering Yes runs the (fake) `psql`, and the audit has an `ask` record.
- `run warn-decline`: same prompt; answering No means `psql` never runs.

`Bash(psql:*)` is in the case's allowed tools, so the only thing that can raise
the prompt is paws.

## Gotchas

- `claude -p` has no human to answer `warn`; only the interactive cases prove it.
- Do not assert that the word "paws" is absent from tool results: `git status`
  prints the untracked `.paws/` directory.
