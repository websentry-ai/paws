---
name: datadog
description: 'Mandatory guardrails for every Datadog write operation (monitor mute/unmute, monitor create/update/delete, downtime, dashboard create/update/delete, incident updates) via the Datadog API or dogshell. Enforces the observability baseline with Datadog mechanics: expiring mutes, fetch-then-diff monitor edits, naming and team/owner tags, confirmed shared deletes, and sanctioned incident flow. Triggers PROACTIVELY on any Datadog write call — do NOT wait for the user to invoke it.'
---

# Datadog

Datadog mechanics for the observability baseline (O1–O5). Apply automatically on every Datadog write — never skip. Primary surface is the API via curl against `https://api.{{DD_SITE}}`; use dogshell (`dog`) where it is cleaner. Every curl below implies the auth headers — pass keys from the environment, **never echo or print their values**:

    -H "DD-API-KEY: ${DD_API_KEY}" -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" -H "Content-Type: application/json"

## Mutes expire (implements O1)

`end` is mandatory — compute the epoch, at most {{MAX_MUTE_DURATION}} from now, and scope the mute as narrowly as the problem allows:

    curl -X POST "https://api.{{DD_SITE}}/api/v1/monitor/<id>/mute" \
      -d '{"end": <epoch>, "scope": "host:<host>"}'

Asked to mute with no end ("just silence it") → apply the cap and say so in one line: "Muting until <time> (org cap {{MAX_MUTE_DURATION}}), reason: <reason>." Explicit insistence on longer → state the risk in one line and proceed; the audit log records it. Record the reason by appending `[muted until <time> by ai-agent: <reason>]` to the monitor message, or in the downtime's `message` field. Downtimes (`POST /api/v1/downtime`) follow the same rule: `"end"` is never omitted. dogshell: `dog monitor mute <id> --end <epoch>` — never bare `dog monitor mute <id>`. Before unmuting, GET the monitor and state who muted it and the recorded reason in one line, then lift it.

## Fetch-then-diff monitor edits (implements O2)

Never PUT a monitor from memory. Fetch, diff, then send the **full** definition back (PUT replaces the object — a partial body silently zeroes options):

    # 1. Fetch current definition
    curl -s "https://api.{{DD_SITE}}/api/v1/monitor/<id>"

    # 2. State the delta before executing:
    #    threshold: 90% → 95% | query: unchanged | recipients: @pagerduty-web removed

    # 3. PUT the complete updated definition
    curl -X PUT "https://api.{{DD_SITE}}/api/v1/monitor/<id>" -d @updated-monitor.json

Any change that removes an `@` recipient or notification channel from `message` → name it in the stated delta ("removing @pagerduty-web from the alert"), then proceed. dogshell (`dog monitor update`) replaces the query/message wholesale — only use it after the same fetch-and-diff.

## Creation carries ownership (implements O3)

Every monitor the agent creates is named per convention and tagged with a team, a human owner, and agent attribution:

    curl -X POST "https://api.{{DD_SITE}}/api/v1/monitor" -d '{
      "type": "metric alert",
      "query": "<query>",
      "name": "{{MONITOR_NAMING}}",
      "message": "<what to do when this fires> @<channel>",
      "tags": ["team:<team>", "owner:<owner>", "created-by:ai-agent"]
    }'

Same three tags on every dashboard (`description` carries the attribution where tags don't apply). A monitor with no recipient in `message` is a monitor nobody sees — always include the notify handle.

## Shared assets (implements O4)

Modifying a dashboard the agent did not create this session — clone-and-edit is the recommended road:

    # 1. Fetch the original
    curl -s "https://api.{{DD_SITE}}/api/v1/dashboard/<id>" > dash.json
    # 2. Edit the copy; retitle "<original title> (edited by ai-agent for <user>)"; POST as new
    curl -X POST "https://api.{{DD_SITE}}/api/v1/dashboard" -d @dash-edited.json

Editing the original in place is fine when the user asks — state whose dashboard it is (`author_handle`) in one line first. Deleting a monitor or dashboard: fetch it first and narrate usage — `author_handle`, notification recipients, `overall_state` and when it last triggered (`GET /api/v1/monitor/<id>` shows `overall_state_modified`). State that, get confirmation, then:

    curl -X DELETE "https://api.{{DD_SITE}}/api/v1/monitor/<id>"

Never `dog monitor delete` or dashboard DELETE as a side effect of "cleaning up" — each deletion is named and confirmed individually.

## Incident hygiene (implements O5)

Incident state changes and status updates go through {{INCIDENT_TOOL}}. If that is Datadog Incident Management, use the v2 API and state severity/state transitions in one line before executing:

    curl -X PATCH "https://api.{{DD_SITE}}/api/v2/incidents/<id>" \
      -d '{"data": {"type": "incidents", "id": "<id>", "attributes": {"fields": {"state": {"type": "dropdown", "value": "stable"}}}}}'

Never move an incident to `resolved`/closed without the required followup attached: {{POSTMORTEM_POLICY}}. Missing link → draft the resolution update, state what's missing, and hand back to the human. If {{INCIDENT_TOOL}} is not Datadog, do not edit incident state via the Datadog API at all — draft the update for the sanctioned tool instead.

## Escalation

Rule vs user intent: state the risk and the compliant path in one line, then proceed as asked — the override is audit-logged. Only deleting or overwriting shared assets still needs a confirmed yes.

## Pre-flight checklist (before every Datadog write)

- [ ] Mute/downtime payload has `"end"` within {{MAX_MUTE_DURATION}}; reason recorded; scope narrowed
- [ ] Monitor edit: current definition fetched; delta stated with recipient removals named; full body on PUT
- [ ] Create: name per `{{MONITOR_NAMING}}`; `team:`/`owner:`/`created-by:ai-agent` tags; notify handle in message
- [ ] Shared delete/overwrite confirmed with usage narrated; keys never echoed
- [ ] Incident change via {{INCIDENT_TOOL}}; no close without the required postmortem link ({{POSTMORTEM_POLICY}})

## Category baseline: Observability

These rules apply to **every** tool that mutates monitors, alerts, dashboards, or incidents. An instance skill (Datadog, Grafana, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. Observability config is the org's sensory system: a mute without an end date is how outages hide, and an unowned monitor is noise nobody answers. Routine edits are narrated and done; only destroying shared assets stops for a confirmed yes.

### Rule O1: Silences always expire

Muting a monitor, alert, or notification channel carries an expiry and a reason:

1. Every mute/downtime carries an end time — {{MAX_MUTE_DURATION}} maximum from now.
2. No end given ("just silence it") → apply the {{MAX_MUTE_DURATION}} cap and say so in one line. Explicit insistence on longer → state the risk in one line and proceed; the audit log records it.
3. Record why: the reason lands in the mute payload, monitor message, or change note per instance mechanics.
4. Unmuting a monitor someone else muted → state who muted it and why in one line, then lift it.

### Rule O2: Alert changes are diffed

Before modifying any monitor or alert rule, fetch its current definition and state the before/after:

1. GET the current definition first — never edit from memory or from what the user pasted.
2. Narrate the delta explicitly: threshold, query, evaluation window, recipients. A removed notification channel or recipient is called out by name in that delta ("removing @pagerduty-web") — dropping a pager route is invisible until the page that never arrives.
3. Send the full definition back on update where the API replaces the object; a partial payload that zeroes fields is a silent rewrite.

### Rule O3: Monitors and dashboards carry ownership

Everything the agent creates is findable and actionable:

- Name per the org convention (`{{MONITOR_NAMING}}`) — never a bare metric name or "test".
- Team tag and owner on every monitor, dashboard, and alert rule, plus agent attribution (`created-by:ai-agent` or the instance equivalent).
- An unowned monitor is unactionable noise: nobody triages it, nobody dares delete it.

### Rule O4: Shared assets are not casually destroyed

Monitors and dashboards are shared infrastructure, not scratch files:

1. Modifying someone else's dashboard → clone-and-edit is the recommended road (copy, edit with attribution, link the original). Editing in place is fine when the user asks for it — state whose asset it is in one line first.
2. Deleting or overwriting a shared monitor/dashboard requires confirmation with usage stated: who it notifies, when it last fired, who views it.

### Rule O5: Incident hygiene

Incident state is the org's shared truth during a fire — it moves only through the sanctioned path:

- State changes and status updates go through {{INCIDENT_TOOL}}; never via side channels or direct field edits that skip its workflow.
- The agent never closes an incident without the org-required postmortem/followup link attached ({{POSTMORTEM_POLICY}}). Missing link → draft the update, state what's missing, hand back to the human.
- Severity changes are stated in one line before executing: current severity, target, why.

### Pre-flight checklist (run before every observability write)

- [ ] Mute/downtime has an end time within {{MAX_MUTE_DURATION}} and a reason (O1)
- [ ] Current definition fetched; before/after stated with recipient removals named (O2)
- [ ] Name per `{{MONITOR_NAMING}}`; team tag, owner, agent attribution set (O3)
- [ ] Shared delete/overwrite confirmed with usage stated (O4)
- [ ] Incident changes via {{INCIDENT_TOOL}}; no close without the required postmortem link (O5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<incident-tool>`: No safe default; the sanctioned incident tool is org-specific. Until set, draft the incident update, name the gap, and let the human file it. Do not change incident state via the Datadog API.
