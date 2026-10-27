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

## Gotchas

- `warn` has no live case. In `claude -p` there is no human to answer the
  approval prompt, so a live run proves nothing about the interactive path.
  The `ask` output is covered by `internal/hook/adapter_test.go`.
- Do not assert that the word "paws" is absent from tool results: `git status`
  prints the untracked `.paws/` directory.
