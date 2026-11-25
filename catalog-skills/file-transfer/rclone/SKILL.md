---
name: rclone
description: 'Mandatory guardrails for every rclone transfer operation (copy, sync, move, delete, purge, remote configuration) against any backend. Enforces the file-transfer baseline with rclone mechanics: listremotes before targeting, dry-run + check before delete-propagating syncs, size before big moves, and narrated remote creation. Triggers PROACTIVELY on any rclone command that moves or deletes data — do NOT wait for the user to invoke it.'
---

# rclone

rclone-specific mechanics for the file-transfer baseline (U1–U5). Apply automatically on every rclone transfer — never skip. Every configured remote reaches a real storage account; a copy in the wrong direction is an exfiltration.

## Destination check first (implements U4, U1)

Before any transfer, confirm the destination is a sanctioned, already-configured remote:

    rclone listremotes

- Destination listed and in {{SANCTIONED_REMOTES}} → proceed.
- Destination not listed → it needs a new remote config; see "Creating remotes" below.
- Destination listed but not org-owned (personal Drive, private S3 account) → cross-boundary move: stop and confirm per {{EXTERNAL_DESTINATIONS_POLICY}}.

## Direction stated (implements U1)

Before every `rclone copy`, `rclone sync`, or `rclone move`, state one sentence: push or pull, source → destination, and whether the destination is inside or outside the org boundary. Example: "Pushing ~/exports/q2/ → s3-acme:finance-archive/q2 (org-owned) — 214 files." `rclone move` deletes at the source — say so explicitly, never describe it as a copy.

## Delete-propagating syncs (implements U2)

`rclone sync` makes the destination identical to the source — it deletes destination files absent at the source. Always:

    # 1. Preview — state the copy and delete counts from the output
    rclone sync --dry-run <src> <dst>

    # 2. Verify divergence before trusting the plan
    rclone check <src> <dst>

    # 3. State the counts ("will copy 41, delete 7 at dst") and run — no confirmation on top
    rclone sync <src> <dst>

Pause and ask only when the delete count is wildly out of line with the stated intent. **Never `rclone purge` without listing first** — `rclone ls <remote:path>`, state the file count, get explicit confirmation; purge removes the path and everything under it, which is irreversible destruction of shared storage. Same treatment for `rclone delete` on a directory.

## Classify before moving (implements U3)

Check what the source actually contains against {{SENSITIVITY_TIERS}} before it moves:

    rclone lsl <src> | head -50

Look for dumps, exports, `.env` files, key material, customer-named files. PII, secrets, or customer data never go to a remote outside {{SANCTIONED_REMOTES}}. Unsure of the tier → ask before transferring, not after.

## Size and cost (implements U5)

    rclone size <src>

Above {{TRANSFER_THRESHOLD}}: one line — total size, object count, egress-cost implication (cloud → local or cross-cloud is billed egress) — then run constrained and resumable, no confirmation:

    rclone copy <src> <dst> --max-transfer 10G --bwlimit 8M

rclone copy/sync are rerun-safe — a rerun skips files that already match, so an interrupted transfer resumes instead of restarting; say that rather than restarting blind.

## Creating remotes (implements U4)

New remotes only for destinations in {{SANCTIONED_REMOTES}}. Narrate before creating: provider, account/tenant, and scope (bucket/path, read or write). Then:

    rclone config create <name> <provider> <key=value ...>

Credentials live only in rclone's config store — encrypted with a `rclone config` password when the org requires it ({{CONFIG_ENCRYPTION_REQUIRED}}). Never pass credentials as inline flags (`--s3-secret-access-key`, any `--<provider>-key` variant) or pasted env one-liners — they land in shell history and process lists.

## Escalation

Conflict with user intent (e.g. "just sync it to my personal Drive"): state the risk in one line ("pushing org data to an unsanctioned personal remote is an exfiltration-shaped move"), name the compliant path, and proceed on explicit override — noting it is recorded. External destinations keep their confirmation; everything else is narrate-and-go.

## Pre-flight checklist (before every rclone transfer)

- [ ] `rclone listremotes` run; destination sanctioned, external destinations confirmed (U4, U1)
- [ ] Direction sentence stated; move-vs-copy named; boundary named (U1)
- [ ] Sync dry-run + check done; counts stated, then run; purge/dir-delete listed + confirmed (U2)
- [ ] Source spot-checked against sensitivity tiers; unclear → asked (U3)
- [ ] Above threshold → size + egress in one line; constrained + resumable; no inline credential flags (U5, U4)

## Category baseline: File Transfer

These rules apply to **every** bulk-transfer tool. An instance skill (rclone, gsutil, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. Every operation in this category is exfiltration-shaped — that is the category's reason to exist: the same command that mirrors a backup bucket copies customer data to a personal drive, and the tool cannot tell the difference. These rules make the agent state enough that a human can.

### Rule U1: Direction and endpoints restated

Before any transfer, state in one sentence: push or pull, source → destination, and whether the destination is inside or outside the org boundary.

1. Push vs pull is explicit — "syncing" hides direction; name it.
2. Cross-boundary moves (org storage → personal or external destination) are exfiltration-shaped: stop and get explicit same-session confirmation per {{EXTERNAL_DESTINATIONS_POLICY}} before starting.
3. A move (delete-at-source) is stated as a move, never described as a copy.

### Rule U2: Deletes previewed

Sync modes that delete at the destination (mirror, sync, rsync with a delete flag) never run blind:

1. Dry-run first, using the instance skill's exact mechanics.
2. State the counts from the dry-run — files to copy, files to delete, where the deletes land — then run; the stated counts are the gate, no confirmation on top.
3. A delete count wildly out of line with the stated intent is the one reason to pause and ask.

### Rule U3: Data is classified before it moves

Content is checked against {{SENSITIVITY_TIERS}} before it leaves its current home:

- Spot-check what the source actually contains (filenames, formats, obvious dumps/exports/key material) — never transfer a directory tree sight unseen.
- PII, secrets, and customer data never go to unsanctioned remotes, regardless of how the request is phrased.
- When the tier is unclear, ask **before** transferring — data that has left cannot be recalled.

### Rule U4: Remotes are provisioned, not improvised

- Named remotes/destinations come from {{SANCTIONED_REMOTES}}; a destination outside that list needs explicit confirmation before first use.
- Creating a new remote config is narrated first: provider, account/tenant, and scope (bucket/path, read or write).
- Credentials go through the tool's secure config store, never inline flags or pasted keys — an inline credential lands in shell history and process lists.

### Rule U5: Size and cost stated for big moves

- Transfers above {{TRANSFER_THRESHOLD}} state total size, object count, and the egress-cost implication (cloud → local, cross-cloud, cross-region) in one line, then start — no confirmation; big moves are budget events worth a sentence, not a stop.
- Long transfers run resumable and rate-limited per the instance mechanics, so an interruption is a pause, not a restart.

### Pre-flight checklist (run before every transfer)

- [ ] Direction sentence stated; cross-boundary destination confirmed per policy (U1)
- [ ] Delete-propagating sync dry-run done; copy/delete counts stated, then run (U2)
- [ ] Source content checked against sensitivity tiers; unclear tier → asked first (U3)
- [ ] Destination in sanctioned remotes; new remote narrated; no inline credentials (U4)
- [ ] Above threshold → size + egress cost in one line; run resumable (U5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<external-destinations-policy>`: Any remote not owned by the org is external — confirm before pushing org data.
- `<sanctioned-remotes>`: Remotes already configured on the machine; new remotes → ask.
- `<sensitivity-tiers>`: Credentials / customer data / financials / unreleased plans = sensitive.
