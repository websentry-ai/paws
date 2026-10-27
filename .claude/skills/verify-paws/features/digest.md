# on_miss: digest

## Sub-features

- `require-skill` with `on_miss: digest` and `digest: <file>` (relative to the rule):
  when the skill is not loaded, the deny reason carries the digest, tagged `paws digest (<rule>):`.
- The retry passes once that deny is in the transcript window (`satisfied_by: digest-delivered`);
  no local skill is needed. A compaction resets it.
- A loaded skill still passes first. A missing or >7,000-byte digest falls back to
  load-required, with `digest_error` in the audit record.
- On `session-start` the digest rides as context instead.

## How to get to it (user POV)

Org-tag catalog bindings we own ship a digest so they work without installing the skill.

## Driving it with drive.py

`run digest-tag`. The digest says tags must be annotated with a `DIGEST-OK:`
message. Proof: one deny carrying the tagged digest, then an allow via
digest-delivered, no Skill call, and `v1.0.0` is an annotated tag whose
message starts with the token.

## Gotchas

- Claude Code records the deny as `toolUseResult: "Error: PreToolUse:<Tool> hook
  error: <reason>"`. `DenyDelivered` requires that prefix and the marker at the
  start of the reason (or of a paragraph), so failed-command output that merely
  prints the marker (`"Error: Exit code 1\n..."`) never counts.
