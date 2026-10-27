# advise

## Sub-features

- Allow the call and attach `message` as additional context, tagged `paws (<rule>):`.
- Once per window: skipped while that tag is in the transcript within 20 turns, re-sent after a compaction.
- Audit: first delivery `verdict: allow`; skips carry `satisfied_by: advice-delivered`.

## How to get to it (user POV)

A tool or session-start rule with `action.type: advise` and a `message`.

## Driving it with drive.py

`run advise-once`. Two sequential `git status` calls. Proof: the first audit
record delivers, the second is `advice-delivered`, no deny, and the reply
carries the advised `NOTE-99` token.

## Gotchas

- Parallel tool calls in one assistant message each get the advice: every hook
  runs before Claude Code writes the first `hook_additional_context`
  attachment. The case prompt forces sequential calls for that reason.
