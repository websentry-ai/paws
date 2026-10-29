# datadog digest

Datadog write: monitor mute/unmute/create/update/delete, downtime, dashboard, incident. Apply before retrying.

API base `https://api.datadoghq.com` (or the org's site). Auth from the environment only: `-H "DD-API-KEY: ${DD_API_KEY}" -H "DD-APPLICATION-KEY: ${DD_APP_KEY}"`. Never echo or print key values.

## Mutes expire
- Every mute and downtime carries `"end"` (epoch), at most 24h from now; scope as narrowly as possible:
  `curl -X POST ".../api/v1/monitor/<id>/mute" -d '{"end": <epoch>, "scope": "host:<host>"}'`
- No end given ("just silence it"): apply the 24h cap and say so: "Muting until <time> (org cap 24h), reason: <reason>." Insistence on longer: state the risk in one line, proceed.
- Record the reason: append `[muted until <time> by ai-agent: <reason>]` to the monitor message, or the downtime `message`.
- dogshell: `dog monitor mute <id> --end <epoch>`, never bare `dog monitor mute <id>`.
- Before unmuting: GET the monitor, state who muted it and why, then lift it.

## Fetch-then-diff monitor edits
1. `curl -s ".../api/v1/monitor/<id>"`: never edit from memory or a pasted copy.
2. State the delta: threshold, query, window, recipients. Name any removed `@` recipient or channel ("removing @pagerduty-web").
3. PUT the full definition (`-d @updated-monitor.json`); a partial body zeroes options. `dog monitor update` only after the same fetch and diff.

## Creation carries ownership
- Name `[<team>] <service> — <condition>`, never a bare metric name or "test".
- Tags `team:<team>`, `owner:<owner>`, `created-by:ai-agent` on every monitor and dashboard (dashboard `description` carries attribution where tags don't apply).
- `message` includes what to do and a notify handle (`@<channel>`).

## Shared assets
- Editing a dashboard the agent didn't create this session: clone-and-edit (fetch, retitle "<title> (edited by ai-agent for <user>)", POST as new). In-place only when the user asks; state the owner (`author_handle`) first.
- Delete or overwrite of a monitor or dashboard: fetch first, state `author_handle`, recipients, `overall_state` and when it last triggered, get confirmation, then DELETE. Each deletion named and confirmed individually; never as "cleanup".

## Incidents
- State changes go through `<incident-tool>`. No incident tool configured: draft the update, name the gap, let the human file it; do not edit incident state via the Datadog API.
- If it is Datadog Incident Management: v2 API (`PATCH .../api/v2/incidents/<id>`); state current and target severity/state and why before executing.
- Never resolve/close without the required postmortem link (SEV-2 and above require a postmortem link before close). Missing: draft, state what's missing, hand back.

## Escalation
Rule vs user intent: state the risk and the compliant path in one line, then proceed. Only shared deletes/overwrites need a confirmed yes.

## Checklist
- [ ] Mute/downtime `"end"` within 24h; reason recorded; scope narrowed
- [ ] Edit: current definition fetched; delta stated, recipient removals named; full body on PUT
- [ ] Create: name per convention; team/owner/created-by tags; notify handle
- [ ] Shared delete/overwrite confirmed with usage; keys never echoed
- [ ] Incident via `<incident-tool>`; no close without the postmortem link
