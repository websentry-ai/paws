# notion digest

Notion write: create/update/move/duplicate pages, create/update databases, comments. Apply before retrying.

## Placement: no orphan pages
1. `notion-create-pages` always has a `parent` (page_id or data_source_id). Never a workspace-level page.
2. Predict the destination from content type, state it ("Creating this under **<parent>**."), proceed:
   - Engineering (reports, architecture, specs) → the teamspace best matching engineering
   - Sales, outreach, demos → the teamspace best matching sales/GTM
   - Product briefs, PRDs, roadmaps → the teamspace best matching product
   - Company-wide updates → the workspace's top-level home/company page
   - Sensitive/strategic (investor updates, financials, hiring) → `<sensitive-space>`; never placed without an explicit destination. Not configured → ask.
3. Ask only when no reasonable prediction exists.
4. After creation, verify the page is nested under the parent or mentioned (`<mention-page>`) from an existing page.

## Ownership
Set the **Owner** person property (or the database row's owner field); no such property → `Owner: @person` at the top of the body. Default owner: the requesting human, never "Claude" or a service account. Created for someone else → set them and say so. Long-lived docs may also tag a maintainer.

## Templates
- PRD → duplicate `<prd-template-page>` and fill its sections. None registered → free-form, and say so.
- Other types → the matching template in `<template-registry>`. Unregistered → free-form, stated: "No org template for this type; writing free-form."

## Changelog on every content edit
On first creation, end the page with:
```
---
### Changelog
| Version | Date | Summary |
|---------|------|---------|
| v1.0 | YYYY-MM-DD | Initial creation |
```
On every content edit (`notion-update-page`, replace-content):
1. Fetch the page, read the changelog.
2. Bump minor (v1.1…) for edits/additions/formatting; major (v2.0…) for structural or >50% rewrites, or when the user says "new version".
3. Append one row with a concise summary.
4. Legacy page without a changelog → add one (v1.0 existing content, v1.1 this edit).
No bump for comments, property-only changes, reads.

## Sensitivity
Credentials, customer data, financials and unreleased plans go only under `<sensitive-space>`, never in team or company-wide spaces. Tier ambiguous → ask before creating. Never create first and move later (page history keeps the original exposure).

## Checklist
- [ ] `parent` set; destination stated or given
- [ ] Owner property/line set to a human
- [ ] Org template used if the type is registered; else free-form stated
- [ ] Changelog present; this edit's row added
- [ ] Sensitive content only under `<sensitive-space>`
