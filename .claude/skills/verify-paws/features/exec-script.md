# exec-script

## Sub-features

- Run `script:` from the rule's own directory with the event JSON on stdin.
- Exit 2 denies; stderr (else stdout) is the reason shown to the agent.
- Any other exit, a timeout (5 s) or a crash allows (fail-open).

## How to get to it (user POV)

A rule with `action.type: exec-script` and an executable script beside it.

## Driving it with drive.py

`run exec-script-commit`. The script denies any commit without a `PAWS-<n>`
ticket. Proof: the deny reason reached the agent, the audit has the deny, and
the final commit subject contains `PAWS-101`.

## Gotchas

- Claude Code labels the reason `PreToolUse:Bash hook error:` in the tool
  result. The agent still treats it as an instruction.
- The prompt must supply the ticket number; without it the agent stops and
  asks rather than inventing one.
- Scripts must be executable in the case tree (`chmod +x`); copytree keeps the mode.
