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

`interactive.py run require-skill-tui` runs the same flow in the real TUI: the
deny and "Successfully loaded skill" render on screen and no permission prompt
appears.

## Gotchas

- If the skill description matches the task, the agent invokes it before the
  first attempt and the deny never fires. The audit then shows only the allow.
- `run compaction-reset` forces a compaction with a `/compact` followup on the
  same session: the commit after it is denied once more. `run resume-no-redeny`
  is its control: the same followup without `/compact` is never denied again.
  The 20-turn window is covered by Go tests (`internal/gate/transcript_test.go`).
- Claude Code can flush the user/assistant records of a step after the next
  tool call's hook ran, so the skill's result may exist only as an
  `api-request-blob` record when the retry is gated. paws reads both; seen
  live in the interactive TUI (`require-skill-tui`), 1 in 3 runs under load
  before the fix.
