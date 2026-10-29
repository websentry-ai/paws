# salesforce digest

Salesforce write: `sf data` create/update/upsert/delete/import, `sf apex run`, or a Salesforce MCP record write. Apply before retrying. Add `--target-org <sf-org>` to every command (`<sf-org>` default: the CLI's target org, `sf config get target-org`; say which).

## Dedup before create (Account, Contact, Lead, Opportunity)
- People: `sf data query --query "SELECT Id, Name, Email, Account.Name FROM Contact WHERE Email = '<email>'"`
- Accounts: `... FROM Account WHERE Website LIKE '%<domain>%'`, then `Name LIKE '%<fragment>%' LIMIT 10`
- Opportunities: `... FROM Opportunity WHERE AccountId = '<id>' AND IsClosed = false`
- Exact match → `sf data update record` on that Id. Near-match → use the obvious one and state it; ask only if ambiguous. No match → create, report what you searched.

## Schema before the first write to an sobject this session
- `sf sobject describe --sobject <Object> --json`
- Fill every field with `"nillable": false` and no default, plus org-mandatory fields in `<required-fields-registry>` (default: schema-required only; say so).
- Picklists verbatim from `picklistValues[].value` (StageName, Industry, LeadSource). Unknown required value → ask; never "TBD".

## Never fabricate
Values only from session sources. Unknown → omit the field from `--values`. Enriched/inferred → append `[source: <where>]` to Description. No emails guessed from name patterns; no Amount/CloseDate copied from sibling opportunities.

## Stage changes
1. Read: `SELECT Id, StageName, Amount FROM Opportunity WHERE Id = '<id>'`
2. Move one valid stage (or the user's named target) with `sf data update record --sobject Opportunity --record-id <id> --values "StageName='<next>'"`.
3. Closed Won / Closed Lost: restate and confirm first — "Moving <name> ($<amount>) from <current> to Closed Won — confirm?" Other moves: one line, then execute.
4. Log the reason: Task with `WhatId='<opp-id>' Subject='Stage change: <from> → <to>' Description='<reason> [created-by: ai-agent]' Status='Completed' OwnerId='<human-owner-id>'`.

## Attribution, deletes, bulk, apex
- Agent-created records: `LeadSource='AI Agent'` where the field exists; append `[created-by: ai-agent]` to Description.
- `OwnerId` is a human (`<record-owner>`: the requesting human; look up `SELECT Id FROM User WHERE Email = '<owner-email>'`). Never the integration user.
- Never `sf data delete record` unprompted. Prefer marking the record inactive/archived via the org's standard flag. If the user insists: state what is lost (activity history, opportunity associations), get explicit confirmation, then proceed.
- Bulk (`import bulk`, `upsert bulk`, or more than 10 records): narrate object, count, fields; run a 5-row sample, state the result, then the full run.
- `sf apex run` with DML is a write: state what it mutates before running.

## User overrides a rule
State the risk in one line and proceed; the override is audit-logged. Only record deletion keeps its confirmation.

## Checklist
- [ ] Dedup SOQL run for creates; result stated
- [ ] `sf sobject describe` this session; required fields filled; picklists verbatim
- [ ] Values session-sourced; unknowns omitted
- [ ] Stage read; closes confirmed, other moves stated; reason Task logged
- [ ] LeadSource + attribution; human OwnerId; no unconfirmed delete; bulk narrated + sampled
