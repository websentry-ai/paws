# session-start

## Sub-features

- `event: session-start` fires on SessionStart (registered by `paws init`).
- `matcher.command_regex` matches the start source: `startup`, `resume`, `clear` or `compact`.
- `require-skill` adds context naming the skill (a start cannot be denied); nothing when already loaded.
- Tool matchers are rejected at parse time.

## How to get to it (user POV)

A rule with `event: session-start`, e.g. `require-skill` of a discipline skill.

## Driving it with drive.py

`run session-start-require-skill`. No tool gate exists, so the only way the
commit gets the skill's `START-7Q:` prefix is the start context making the
agent load the skill. Proof: Skill invoked, audit has the session-start
record, no deny anywhere, commit carries the prefix.

## Gotchas

- `paws explain --event session-start startup` dry-runs it.
- After `compact` the skill text is gone; a `compact` start fires the context again.
