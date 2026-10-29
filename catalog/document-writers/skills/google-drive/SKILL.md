---
name: google-drive
description: 'Mandatory guardrails for every Google Drive write operation (create_file, copy_file) via the Google Drive MCP server. Enforces the document-writers baseline with Drive mechanics: parent-folder placement on create, human ownership noted for shared-drive files, template copies from the org registry, version notes on substantive edits, and a permission check before sensitive content lands in broadly-shared folders. Triggers PROACTIVELY on any Google Drive MCP write — do NOT wait for the user to invoke it.'
---

# Google Drive

Drive-specific mechanics for the document-writers baseline (D1–D5). Apply automatically on every Drive write — never skip. Primary surface is the Google Drive MCP server (`google-drive`; some clients register it as `Google_Drive`): `search_files` / `read_file_content` / `get_file_permissions` for checks, `create_file` / `copy_file` for writes.

## Placement (implements D1)

Every `create_file` sets a parent folder — never My Drive root, never unfiled.

1. Predict the parent from content type using {{DRIVE_FOLDER_MAP}}; locate it with `search_files` (query: `name contains '<folder>' and mimeType = 'application/vnd.google-apps.folder'`).
2. State it in one line — "Creating this in **<folder>**." — and proceed. Ask only when no reasonable prediction exists.
3. After creation, verify the file actually sits under that parent (the create response's parent, or `get_file_metadata`).

## Ownership (implements D2)

The owner is the human whose Drive it is — never the connected service account, never "Claude".

- My Drive files: Drive records the connected human as owner automatically; nothing extra needed.
- Shared-drive files have no per-file owner — put an `Owner: <person>` line at the top of the document header, defaulting to {{DEFAULT_OWNER}}.
- Created for someone else → name them as owner in that header line and say so.

## Templates (implements D3)

Registered doc type (PRD, design doc, postmortem, runbook, meeting notes) → `copy_file` the org template from {{TEMPLATE_REGISTRY}} into the target folder, then fill its sections — never rebuild a sanctioned structure from scratch with `create_file`. Unregistered type → free-form is fine, but say so: "No org template for this type; writing free-form."

## Version notes (implements D4)

Drive keeps automatic revision history, so the changelog burden is lighter than Notion's — no changelog table needed. On a **substantive** update (content rewritten, sections added or removed, meaning changed — not typo or formatting fixes), append one version-note line at the document's end:

    v1.1 — YYYY-MM-DD — <one-line summary of what changed and why>

Revision history answers "what changed"; the version note answers "why". Property-only changes (rename, move, star) and comments get no note.

## Sensitivity and sharing (implements D5)

Before writing sensitive content ({{SENSITIVITY_TIERS}}) into any folder or existing file:

1. Run `get_file_permissions` on the destination.
2. Destination broadly shared ({{BROAD_ACCESS_BOUNDARY}}) → do not write there; place it in {{SENSITIVE_DRIVE}} instead. Sensitive content is never placed on a prediction — the destination must be explicit or configured.
3. Sharing changes that would make a file public or org-wide are exposure-class: state exactly who gains access and **confirm before applying** — one of the few actions that always confirms.

## Escalation

When a rule conflicts with user intent (e.g. "just drop the customer list in the team folder"): state the risk in one line ("that folder is domain-visible — every employee could read the customer list"), offer the compliant path, and proceed on explicit override — stating that the override will be recorded.

## Pre-flight checklist (before every Drive write)

- [ ] Parent folder set and stated; never My Drive root (D1)
- [ ] Owner is a human; shared-drive files carry an `Owner:` header line (D2)
- [ ] Registered doc type → `copy_file` from the template; else free-form stated (D3)
- [ ] Substantive edit → version note appended; Drive revisions cover the rest (D4)
- [ ] `get_file_permissions` checked before sensitive content; public/org-wide sharing changes confirmed (D5)

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

- `<sensitive-drive>`: No default. Sensitive content is never placed without an explicit destination — ask the user where it goes.
- `<template-registry>`: No default — free-form is allowed. Write free-form and say no org template is registered.
