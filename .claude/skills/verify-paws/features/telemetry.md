# Local telemetry

## Sub-features

- Audit schema v2 (`"v": 2` on every record) in `$PAWS_HOME/audit.log`, rotated to `audit.log.1` past 50 MB.
- Count-only records: `skill_invoked` (PreToolUse on Skill), `tool_result` (PostToolUse, tool name), `turn_end` (Stop).
- The Activity tab of `paws ui` (`/api/stats`): per rule fires, denied, already-loaded, loaded-after-deny, delivered; dormant rules; skills; totals.

## How to get to it (user POV)

Nothing to configure after `paws init`. Open `paws ui` and pick Activity.

## Driving it with drive.py

`run require-skill-commit`: the audit sequence includes `skill_invoked` for the
skill, then the allow, then `tool_result` for Bash and `turn_end`.

## Gotchas

- Telemetry records never hold command, prompt or file text; rule records still hold the redacted target.
- Re-run `paws init` after upgrading: PostToolUse and Stop are new registrations.
