# google-drive digest

Google Drive MCP `create_file` or `copy_file`. Apply before retrying. Checks: `search_files`, `read_file_content`, `get_file_metadata`, `get_file_permissions`.

## Placement: no orphan files
1. Every `create_file` sets a parent folder. Never My Drive root, never unfiled.
2. Predict the parent from content type using the existing folder and shared-drive names; find it with `search_files` (`name contains '<folder>' and mimeType = 'application/vnd.google-apps.folder'`).
3. State it — "Creating this in **<folder>**." — and proceed. Ask only when no reasonable prediction exists.
4. After creation, verify the file sits under that parent (create response or `get_file_metadata`).

## Ownership
- Owner is the human whose Drive it is (default: the requesting human), never the service account or "Claude".
- My Drive: Drive records the connected human automatically.
- Shared drive: put an `Owner: <person>` line at the top of the document.
- Created for someone else → name them as owner and say so.

## Templates
- PRD, design doc, postmortem, runbook, meeting notes: `copy_file` the org template from `<template-registry>` into the target folder and fill its sections; never rebuild a sanctioned structure with `create_file`.
- No registry configured or no matching template → free-form, and say so: "No org template for this type; writing free-form."

## Version notes
Drive keeps revision history, so no changelog table. On a substantive update (content rewritten, sections added/removed, meaning changed), append one line at the end:
`v1.1 — YYYY-MM-DD — <what changed and why>`
No note for typo/format fixes, rename, move, star, or comments.

## Sensitivity and sharing
Sensitive = credentials, customer data, financials, unreleased plans.
1. Before writing sensitive content into a folder or file: `get_file_permissions` on the destination.
2. Broadly shared (anyone-with-link or domain-wide: permission type `anyone` or `domain`) → don't write there; use `<sensitive-drive>`. No sensitive drive configured → ask where it goes. Sensitive content is never placed on a prediction.
3. A sharing change that makes a file public or org-wide: state exactly who gains access and confirm before applying. Always confirms.
When in doubt about the tier, ask before creating.

## User overrides a rule
One line of risk ("that folder is domain-visible — every employee could read the customer list"), offer the compliant path, proceed on explicit override and say it will be recorded.

## Checklist
- [ ] Parent folder set and stated; not My Drive root
- [ ] Human owner; shared-drive files carry an `Owner:` line
- [ ] Registered doc type → `copy_file` from the template; else free-form stated
- [ ] Substantive edit → version note
- [ ] Permissions checked before sensitive content; public/org-wide sharing confirmed
