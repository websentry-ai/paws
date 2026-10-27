# require-skill gate

## Sub-features

- Deny a matching call until the named skill was invoked with the Skill tool.
- Allow once the transcript shows a successful invocation in the last 20 assistant turns.
- Never deny again while the skill is loaded (no loop).
- A compaction boundary resets the window.
- Subagent (sidechain) invocations do not count.

## How to get to it (user POV)

Drop a rule with `action.type: require-skill` and `skill: <name>` in `.paws/`,
run `paws init`, ask Claude for the gated action.

## Driving it with drive.py

`run require-skill-commit`. The skill's description is deliberately unrelated
to commits so the agent does not preload it. Proof: audit shows exactly one
`deny` then an `allow` with `satisfied_by: skill-load`, the Skill tool was
invoked, and the commit subject carries the skill's `PROBE-7Q:` token.

## Gotchas

- If the skill description matches the task, the agent invokes it before the
  first attempt and the deny never fires. The audit then shows only the allow.
- The compaction and 20-turn reset are covered by Go tests
  (`internal/gate/transcript_test.go`), not by a live case: forcing a
  compaction headlessly is slow and nondeterministic.
