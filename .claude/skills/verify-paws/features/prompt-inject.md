# prompt-inject

## Sub-features

- On `user-prompt` rules only: append `message:` to the agent's context for that prompt.
- Optional `command_regex` matches the prompt text.
- Not written to the audit log.

## How to get to it (user POV)

A rule with `event: user-prompt` and `action.type: prompt-inject`.

## Driving it with drive.py

`run prompt-inject`. The injected house rule tells the agent to end with
`ZEBRA-42`. Proof: the final reply contains the token.

## Gotchas

- There is no audit record to assert on by design; the token in the reply is
  the only evidence.
