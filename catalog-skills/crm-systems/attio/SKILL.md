---
name: attio
description: 'Mandatory guardrails for every Attio write operation (create-record, update-record, upsert-record, create-note, create-task, list-entry updates) via the Attio MCP server. Enforces the crm-systems baseline with Attio mechanics: search-records dedup before create, list-attribute-definitions before first write, verbatim select values, one-line state-and-proceed stage moves, and agent attribution on notes. Triggers PROACTIVELY on any Attio MCP write — do NOT wait for the user to invoke it.'
---

# Attio

Attio mechanics for the crm-systems baseline (R1–R5). Primary surface is the Attio MCP server — invocations below are MCP tool names with their key parameters. Apply on every write: narrate in one line and proceed; ask only where a rule says ask; confirm nothing that Attio can recover.

## Dedup search (implements R1)

Before any `create-record`, search on the org's match fields ({{DEDUP_MATCH_FIELDS}}):

    # People — match on email
    search-records { object: "people", query: "<email>" }

    # Companies — match on domain, then name fragment
    search-records { object: "companies", query: "<domain>" }
    search-records { object: "companies", query: "<name-fragment>" }

    # Deals — open deals on the same company
    list-records { object: "deals", filter: <company + open stage> }

Exact match → `update-record { object, record_id: "<existing>" }` instead of creating. Near-match → show the candidates and ask "use existing or create new". No match → create, stating what you searched in one line.

## Data integrity (implements R2)

Every attribute value traces to a session source (user statement, document, email, enrichment output). Unknown email, phone, title, deal value → omit the attribute from the `create-record`/`update-record` payload entirely — never guess, never copy from a sibling record. Enriched or inferred values get their origin noted in a `create-note` on the record: `[source: <where>]`.

## Schema before write (implements R3)

Before the first write to an object this session:

    list-attribute-definitions { object: "<object-slug>" }
    # list writes: list-list-attribute-definitions { list_id }

Fill every org-required attribute at creation ({{REQUIRED_FIELDS_REGISTRY}}); select/status option values verbatim from the definition's options — never invent a stage, status, or category. Unknown required value → ask, never "TBD".

## Stage transitions (implements R4)

1. `get-records-by-ids { object: "deals", record_ids: ["<id>"] }` — read the current stage first.
2. `update-record { object: "deals", record_id, values: { stage: "<verbatim option>" } }` — state the move in one line and proceed: "Moving Acme renewal Discovery → Evaluation."
3. Closing (won or lost) or skipping stages: one-line statement with deal name and value before the write — "Closing Acme renewal ($40k) as Won, reason: signed 2026-07-18" — then record the reason via `create-note` on the deal.

## Attribution, deletes, lists (implements R5)

- Every agent-created record gets a `create-note` carrying {{AGENT_ATTRIBUTION_CONVENTION}}; task assignees and record owners are humans ({{DEFAULT_RECORD_OWNER}}) resolved via `list-workspace-members` — never the integration user.
- No record deletion without a one-line statement of what goes with it (notes, list entries, activity). Prefer {{ARCHIVE_PATH}}; Attio archives are recoverable, so a stated delete-to-archive proceeds without a confirmation round-trip. `delete-comment` only on the agent's own comments.
- `add-record-to-list` / `update-list-entry-by-record-id` above {{BULK_THRESHOLD}} records: state count, list, and attributes changed in one line first, then proceed.

## Escalation

Rule conflicts with user intent ("import all 80 contacts, skip the search"): one line of risk ("80 unswept creates will fork existing people records and split activity history"), offer the compliant path (search the 80 emails first, create only the misses), proceed on explicit override — noting the override is recorded.

## Pre-flight checklist (before every Attio write)

- [ ] `search-records` dedup run for creates (email/domain); result stated in one line (R1)
- [ ] Every value session-sourced; unknowns omitted from the payload; enrichment noted `[source: …]` (R2)
- [ ] `list-attribute-definitions` done this session for the object; required attributes filled; select values verbatim (R3)
- [ ] Stage change: current stage read; move/close stated in one line with a reason note (R4)
- [ ] Attribution note added; owner is a human; deletes and bulk list ops stated before proceeding (R5)

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

- `<record-owner>`: The requesting human, resolved via `list-workspace-members`. If it is unclear who that is, ask the user.
- `<required-fields-registry>`: No default — only the attributes the schema itself marks required. Fill those and say no org registry is configured.
