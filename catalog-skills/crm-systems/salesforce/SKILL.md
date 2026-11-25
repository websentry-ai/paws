---
name: salesforce
description: 'Mandatory guardrails for every Salesforce write operation (record create, update, upsert, delete, bulk import, apex execution) via the sf CLI or Salesforce MCP servers. Enforces the crm-systems baseline with Salesforce mechanics: SOQL dedup queries, sobject describe before write, StageName transition rules, and agent attribution. Triggers PROACTIVELY on any `sf data` write or apex run — do NOT wait for the user to invoke it.'
---

# Salesforce

Salesforce-specific mechanics for the crm-systems baseline (R1–R5). Apply automatically on every Salesforce write — never skip. All commands target {{SF_ORG_ALIAS}}; add `--target-org {{SF_ORG_ALIAS}}` to every invocation shown below.

## Dedup search (implements R1)

Before any `sf data create record` on Account, Contact, Lead, or Opportunity, run the matching query and state the result:

    # People (Contact/Lead) — match on email
    sf data query --query "SELECT Id, Name, Email, Account.Name FROM Contact WHERE Email = '<email>'"

    # Companies (Account) — match on website domain, then fuzzy name
    sf data query --query "SELECT Id, Name, Website FROM Account WHERE Website LIKE '%<domain>%'"
    sf data query --query "SELECT Id, Name FROM Account WHERE Name LIKE '%<name-fragment>%' LIMIT 10"

    # Opportunities — match on account + open stage
    sf data query --query "SELECT Id, Name, StageName FROM Opportunity WHERE AccountId = '<id>' AND IsClosed = false"

Exact match → `sf data update record` on the existing Id. Near-match → show the candidate rows, use the obvious one, and state the choice; ask only when genuinely ambiguous. No match → create and report what you searched.

## Schema before write (implements R3)

Before the first write to any sobject in a session:

    sf sobject describe --sobject <ObjectName> --json

From the output: fill every field with `"nillable": false` and no default; use picklist values verbatim from `picklistValues[].value` — especially `StageName`, `Industry`, `LeadSource`. Org-mandatory custom fields beyond the schema live in {{REQUIRED_FIELDS_REGISTRY}}. An unknown required value → ask the user; never write "TBD".

## Data integrity (implements R2)

Field values come only from session sources. Unknown → omit the field entirely from the `--values` string. When a value came from enrichment or inference, append its origin to the Description field: `[source: <where>]`. Never guess emails from name patterns; never copy Amount or CloseDate from sibling opportunities.

## Stage transitions (implements R4)

    # 1. Read current stage
    sf data query --query "SELECT Id, StageName, Amount FROM Opportunity WHERE Id = '<id>'"

    # 2. Move one valid stage (or the user's explicitly named target)
    sf data update record --sobject Opportunity --record-id <id> --values "StageName='<next-stage>'"

Closing (any `Closed Won` / `Closed Lost` value) is forecast-affecting: restate and confirm first — "Moving <name> ($<amount>) from <current> to Closed Won — confirm?" Any other stage move, including a skip the user asked for, is stated in one line and executed. Either way, log the reason:

    sf data create record --sobject Task --values "WhatId='<opp-id>' Subject='Stage change: <from> → <to>' Description='<reason> {{AGENT_ATTRIBUTION_CONVENTION}}' Status='Completed' OwnerId='<human-owner-id>'"

## Attribution, deletes, bulk (implements R5)

- Every agent-created record sets `LeadSource='{{AGENT_LEAD_SOURCE}}'` where the field exists, and appends {{AGENT_ATTRIBUTION_CONVENTION}} to Description. `OwnerId` is always a human user Id (default {{DEFAULT_RECORD_OWNER}}) — look it up, never leave the integration user as owner:

      sf data query --query "SELECT Id FROM User WHERE Email = '<owner-email>'"

- **Never run `sf data delete record` unprompted.** Preferred path: {{ARCHIVE_PATH}}. If the user insists: state what is lost (activity history, opportunity associations), get explicit confirmation, then proceed — the override is recorded in the audit log.
- Bulk (`sf data import bulk`, `sf data upsert bulk`, or >{{BULK_THRESHOLD}} records): narrate object, record count, and fields changed; run a 5-row sample, state the result, and proceed to the full run.
- `sf apex run` that performs DML counts as a write: state what the anonymous Apex mutates before executing.

## Escalation

Rule conflicts with user intent → state the risk in one line and proceed; the override lands in the audit log. Only record deletion keeps its confirmation.

## Pre-flight checklist (before every Salesforce write)

- [ ] Dedup SOQL run for Account/Contact/Lead/Opportunity creates; result stated
- [ ] `sf sobject describe` done this session for the target object; required fields filled; picklists verbatim
- [ ] Every value session-sourced; unknowns omitted, not guessed
- [ ] Stage change: current stage read; Closed Won/Lost confirmed, other moves stated; reason task logged
- [ ] LeadSource + attribution set; OwnerId is a human; no delete without confirmation; bulk narrated

## Category baseline: CRM Systems

These rules apply to **every** CRM-writing tool. An instance skill (Salesforce, HubSpot, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. CRM data is the org's revenue memory: a duplicate account splits pipeline history, a fabricated email poisons outreach, a silently skipped stage corrupts forecasting.

### Rule R1: Search before create

Never create an account, contact, lead, deal, or company record without first searching for an existing one:

1. Search on the org's dedup match fields ({{DEDUP_MATCH_FIELDS}} — typically email for people, domain for companies) using the instance skill's exact query mechanics.
2. Exact match → update the existing record instead of creating.
3. Near-match (same domain, similar name) → show the candidates, pick the obvious one, and state the choice; ask only when the match is genuinely ambiguous.
4. No match → create, and say what you searched so the user can spot a miss.

### Rule R2: Never fabricate CRM data

Every value written to a CRM must be traceable to a source in the current session (user statement, document, email, enrichment tool output):

- Unknown email, phone, title, company size, deal value → **leave the field blank**. A blank field is a task; a guessed field is a landmine.
- Never extrapolate ("probably uses the firstname.lastname@ pattern") and never copy values between similar records.
- When a value comes from enrichment or inference, note the source in the record's notes/description per the instance mechanics.

### Rule R3: Respect the schema — required fields and picklists

Before writing, read the object's actual schema via the instance skill's describe mechanics:

- Fill every org-required field at creation time ({{REQUIRED_FIELDS_REGISTRY}}); unknown required value → ask, never stub with "TBD".
- Picklist/select values verbatim from the schema — never invent a stage name, industry, or status that isn't in the list.
- Custom fields the org marked mandatory in {{REQUIRED_FIELDS_REGISTRY}} count as required even when the CRM itself doesn't enforce them.

### Rule R4: Pipeline transitions are deliberate

Stage and status changes drive forecasting and automation — they are never a side effect:

1. Read the record's current stage before changing it.
2. Move only to a valid adjacent stage unless the user explicitly names the target.
3. Closing (won or lost) is forecast-affecting and requires explicit same-session user confirmation, restated before executing: "Moving <deal> from X to Closed Won — confirm?" Any other stage move — including a skip the user asked for — is stated in one line and executed.
4. Record the reason for the transition in the change note/activity per instance mechanics.

### Rule R5: Attribution and no silent destruction

- Every agent-created or agent-edited record carries the org's agent-attribution convention ({{AGENT_ATTRIBUTION_CONVENTION}}) so RevOps can distinguish agent writes.
- Record ownership is always a human ({{DEFAULT_RECORD_OWNER}}), never a service account.
- **Never delete CRM records.** Prefer the org's archive/deactivate path ({{ARCHIVE_PATH}}). If the user insists on deletion, restate what will be lost (activity history, associations) and proceed only on explicit confirmation — the override lands in the audit log.
- Bulk operations (creates/updates above {{BULK_THRESHOLD}} records) get narrated first: count, objects touched, fields changed.

### Pre-flight checklist (run before every CRM write)

- [ ] Dedup search done on match fields; result stated (R1)
- [ ] Every value traceable to a session source; unknowns left blank (R2)
- [ ] Required fields filled; picklist values verbatim from schema (R3)
- [ ] Stage changes: current stage read, transition valid, Closed Won/Lost confirmed, other moves stated (R4)
- [ ] Attribution set; owner is a human; no deletes without confirmed override (R5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<record-owner>`: The requesting human; look up their User Id. Use it and say who owns the record.
- `<required-fields-registry>`: No default — only the fields the schema marks required (`nillable` false, no default). Fill those and say no org registry is configured.
- `<sf-org>`: The CLI's default org (`sf config get target-org`). Use it and say which org you targeted.
