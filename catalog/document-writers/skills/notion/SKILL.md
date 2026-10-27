---
name: notion
description: 'Mandatory guardrails for every Notion write operation (create-pages, update-page, replace-content, create-database). Enforces the document-writers baseline with Notion-specific mechanics: parent/teamspace placement, owner tagging, org templates, and an in-page version changelog. Triggers PROACTIVELY on any Notion MCP tool call that creates or modifies a page — do NOT wait for the user to invoke it.'
---

# Notion

Notion-specific mechanics for the document-writers baseline (D1–D5). Apply automatically on every Notion write — never skip.

## Placement (implements D1)

Before calling `notion-create-pages`:

1. Confirm a `parent` is set (page_id or data_source_id). **Never create a workspace-level page.**
2. Predict the destination from content type, state it, and proceed:

| Content | Destination |
|---------|-------------|
| Engineering (weekly reports, architecture, specs) | {{ENG_TEAMSPACE}} |
| Sales, outreach, demos | {{SALES_TEAMSPACE}} |
| Product briefs, PRDs, roadmaps | {{PRODUCT_TEAMSPACE}} |
| Sensitive/strategic (investor updates, financials, hiring) | {{SENSITIVE_SPACE}} |
| Company-wide updates | {{COMPANY_HOME}} |

> "Creating this under **[predicted parent]**."

Ask only when no reasonable prediction exists — or when the destination is {{SENSITIVE_SPACE}}-tier content, which is never placed without an explicit destination.

3. After creation, verify the page is reachable: nested under the parent, or mentioned (`<mention-page>`) from at least one existing page.

## Ownership (implements D2)

Set the **Owner** person property on the page (or the database row's owner field). If the parent database has no owner property, add an `Owner: @person` line at the top of the page body. Owner defaults to {{DEFAULT_OWNER}}; a maintainer may additionally be tagged for long-lived docs (runbooks, wikis).

## Templates (implements D3)

Check the doc type against the org template registry before writing:

- PRD → duplicate {{PRD_TEMPLATE_PAGE}} and fill its sections; do not invent structure.
- Other registered types → same pattern via {{TEMPLATE_REGISTRY}}.
- Unregistered type → free-form, and say so.

## Version changelog (implements D4)

On **first creation**, end the page with:

```
---
### Changelog
| Version | Date | Summary |
|---------|------|---------|
| v1.0 | YYYY-MM-DD | Initial creation |
```

On **every content edit** (`notion-update-page`, replace-content):

1. Fetch the page and read the current changelog.
2. Bump: **minor** (v1.1, v1.2 …) for content edits/additions/formatting; **major** (v2.0 …) for structural rewrites, >50% rewrites, or when the user says "new version".
3. Append one row with a concise summary.
4. Legacy page with no changelog → add one (v1.0 for existing content, v1.1 for this edit).

No bump for: comments, property-only changes, reads.

## Sensitivity (implements D5)

Before content lands, check it against the sensitivity tiers ({{SENSITIVITY_TIERS}}). Credentials, customer data, financials, and unreleased plans go only under {{SENSITIVE_SPACE}} — never in team or company-wide spaces. When the tier is ambiguous, ask before creating; never create first and move later (Notion page history keeps the original location's exposure).

## Pre-flight checklist (before every Notion write)

- [ ] `parent` set; destination stated or explicitly given
- [ ] Owner property/line set to a human
- [ ] Org template used if the doc type is registered
- [ ] Changelog block present; this edit's row prepared
- [ ] Sensitivity checked — sensitive content only under {{SENSITIVE_SPACE}}

## Category baseline: Document Writers

These rules apply to **every** document-writing tool. An instance skill (Notion, Google Drive, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them.

### Rule D1: No orphan documents

Every document you create MUST be reachable from an existing document or container. Before creating:

1. Identify the parent (page, folder, space, drive) the document belongs under.
2. If no parent is specified, **predict one from the content type, state it in one line, and proceed**. Ask only when no reasonable prediction exists.
3. After creation, verify the document is actually linked/nested under that parent.

Never create a document at the workspace root or in an unfiled location.

### Rule D2: Ownership is explicit

Every document MUST carry an identifiable owner and maintainer:

- Set the owner property/field if the tool supports one; otherwise state the owner in the document header.
- Default owner: the human you are working for ({{DEFAULT_OWNER}}), never "Claude" or a service account.
- If the document is created for someone else, set that person as owner and say so.

### Rule D3: Templates before free-form

If the org has a template for this document type, **use it**. Check the document type against the org template registry ({{TEMPLATE_REGISTRY}}) before writing:

- PRD, design doc, postmortem, runbook, meeting notes → use the matching template's section structure verbatim.
- No matching template → free-form is fine, but say so: "No org template for this type; writing free-form."

Do not invent your own section structure when a sanctioned one exists.

### Rule D4: Every edit is versioned

Treat documents like code. Every content edit gets a changelog entry (version, date, summary) appended to the document, using the instance skill's changelog mechanics. Property-only changes (title, tags, status) and comments do not bump the version.

### Rule D5: Classify before you write

Before writing content, check it against the org's sensitivity tiers ({{SENSITIVITY_TIERS}}). Content containing credentials, customer data, financials, or unreleased plans goes only in locations sanctioned for that tier — when in doubt, ask before creating.

### Pre-flight checklist (run before every create/edit call)

- [ ] Parent identified and stated (D1)
- [ ] Owner set to a human (D2)
- [ ] Org template applied if one exists for this doc type (D3)
- [ ] Changelog entry prepared for this edit (D4)
- [ ] Sensitivity tier checked against destination (D5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<prd-template-page>`: No registered PRD template. Write free-form and say so.
- `<sensitive-space>`: No default. Sensitive content is never placed without an explicit destination — ask the user where it goes.
- `<template-registry>`: No default — free-form is allowed. Write free-form and say no org template is registered.
