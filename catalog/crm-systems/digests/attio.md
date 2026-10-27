# attio digest

Attio MCP write: create/update/upsert record, note, task, list entry. Apply before retrying the call. Narrate in one line and proceed; ask only where a rule says ask.

## Search before create
- People: `search-records { object: "people", query: "<email>" }`. Companies: by domain, then name fragment. Deals: `list-records` for open deals on the same company.
- Exact match → `update-record` on the existing record_id instead of creating.
- Near-match → show candidates, ask "use existing or create new".
- No match → create, stating what you searched in one line.

## Never fabricate
- Every value traces to a session source (user, document, email, enrichment output).
- Unknown email, phone, title, deal value → omit the attribute from the payload. Never guess, never copy from a sibling record.
- Enriched/inferred values: `create-note` on the record with `[source: <where>]`.

## Schema before the first write to an object this session
- `list-attribute-definitions { object }` (lists: `list-list-attribute-definitions { list_id }`).
- Fill every required attribute (`<required-fields-registry>`; default: schema-required only, say no registry is configured). Unknown required value → ask, never "TBD".
- Select/status values verbatim from the definition's options.

## Stage moves
1. `get-records-by-ids` to read the current stage.
2. Move to a valid adjacent stage (or the user's named target), stated in one line: "Moving Acme renewal Discovery → Evaluation."
3. Closing (won/lost) or skipping: one line with deal name and value first ("Closing Acme renewal ($40k) as Won, reason: …"), then a `create-note` with the reason. The category baseline asks for explicit same-session confirmation before Closed Won/Lost: "Moving <deal> from X to Closed Won — confirm?"

## Attribution, deletes, bulk
- Every agent-created record gets a `create-note` carrying `[created-by: ai-agent]`.
- Owners and task assignees are humans (`<record-owner>`: the requesting human, resolved via `list-workspace-members`; ask if unclear). Never the integration user.
- Prefer archiving (remove from active lists, mark archived via the org's status attribute). Any delete: state in one line what goes with it (notes, list entries, activity). The baseline requires explicit confirmation for a real record deletion.
- `delete-comment` only on the agent's own comments.
- List ops or writes over 10 records: state count, list/objects and attributes changed first.

## User overrides a rule
One line of risk, offer the compliant path (e.g. search the 80 emails first, create only the misses), proceed on explicit override and say it is recorded.

## Checklist
- [ ] Dedup search run for creates; result stated
- [ ] Values session-sourced; unknowns omitted; enrichment noted `[source: …]`
- [ ] Attribute definitions read this session; required filled; selects verbatim
- [ ] Stage read first; moves stated; closes confirmed with a reason note
- [ ] Attribution note; human owner; deletes and bulk ops stated
