---
name: gsutil
description: 'Mandatory guardrails for every Google Cloud Storage transfer (cp, rsync, mv, rm) via gsutil or its successor CLI `gcloud storage` — same rules, both command shapes. Enforces the file-transfer baseline with GCS mechanics: du before big copies, rsync -n dry-runs, and org-ownership checks on destination buckets. Triggers PROACTIVELY on any gsutil or `gcloud storage` command that copies, syncs, or deletes objects — do NOT wait for the user to invoke it.'
---

# gsutil / gcloud storage

GCS-specific mechanics for the file-transfer baseline (U1–U5). Apply automatically on every gsutil write — never skip. `gcloud storage` is the successor CLI ({{PREFERRED_CLI}}); every rule below applies identically to both command shapes, and equivalents are shown where flags differ.

## Destination is org-owned (implements U1, U4)

Before pushing to any bucket, verify it belongs to the org:

    gsutil ls -L -b gs://<dest-bucket>
    # gcloud shape: gcloud storage buckets describe gs://<dest-bucket>

Check the project against {{GCS_ORG_PROJECTS}}. A bucket outside {{SANCTIONED_REMOTES}} is a cross-boundary destination — stop and confirm per {{EXTERNAL_DESTINATIONS_POLICY}} before the first byte moves.

Then state the direction sentence before every cp/rsync/mv: push or pull, source → destination, inside or outside the org boundary. Example: "Pushing ./build-artifacts/ → gs://acme-releases/v2.4 (org project acme-prod) — 1.2 GB." `mv` deletes at the source — say so.

## Dry-run syncs (implements U2)

`gsutil rsync` deletes at the destination when `-d` is passed (`gcloud storage rsync --delete-unmatched-destination-objects`). Always:

    # 1. Preview — -n is dry-run; state copy/delete counts from the output
    gsutil -m rsync -n <src> <dst>

    # 2. State the counts, then run — no confirmation on top
    gsutil -m rsync <src> <dst>

Never run `-d` without a same-session dry-run; pause and ask only when the delete count is wildly out of line with the stated intent. `gsutil rm -r` on a prefix gets a listing (`gsutil ls -r gs://<path>`) and a stated object count before it runs.

## Classify before moving (implements U3)

Spot-check the source against {{SENSITIVITY_TIERS}} before it moves:

    gsutil ls -r gs://<src-path> | head -50

Look for dumps, exports, `.env`/key files, customer-named objects. PII, secrets, or customer data never leave for a bucket outside {{SANCTIONED_REMOTES}}. Unsure of the tier → ask before transferring, not after.

## Size and cost (implements U5)

    gsutil du -s gs://<url>
    # gcloud shape: gcloud storage du --summarize gs://<url>

Above {{TRANSFER_THRESHOLD}}: one line — total size, object count, egress implication (GCS → local and cross-cloud are billed egress; intra-region GCS→GCS is not) — then run, no confirmation. Big copies use `-m` for parallelism; gsutil/gcloud uploads are resumable by default, so an interrupted run reruns safely — say that rather than restarting blind.

## Out of scope: IAM and ACLs

`gsutil iam`, `gsutil acl`, `gsutil signurl`, and bucket-policy changes are **not** transfer operations — they are access-control changes governed by the cloud-resources category (aws-s3-style governance) and identity-access. Never "fix" a permission error mid-transfer by widening an ACL or IAM binding; stop and escalate to those skills instead.

## Credentials (implements U4)

Auth flows through `gcloud auth login` / workload identity only. Never embed service-account keys in commands — no `-o Credentials:gs_service_key_file=...`, no `GOOGLE_APPLICATION_CREDENTIALS=<pasted-key>` one-liners, no JSON keys copied into the working tree to make a transfer work. A key file must already exist via the org's provisioning path or the transfer waits.

## Escalation

Conflict with user intent (e.g. "just cp it to my personal bucket"): state the risk in one line ("pushing org data to a bucket outside the org's projects is an exfiltration-shaped move"), name the compliant path, and proceed on explicit override — noting it is recorded. External destinations keep their confirmation; everything else is narrate-and-go.

## Pre-flight checklist (before every gsutil / gcloud storage write)

- [ ] `gsutil ls -L -b` on the destination; org-owned proceeds, external confirmed (U1, U4)
- [ ] Direction sentence stated; mv-vs-cp named; boundary named (U1)
- [ ] `rsync -n` dry-run done; counts stated, then run; `-d` never without dry-run (U2)
- [ ] Source spot-checked against sensitivity tiers; unclear → asked (U3)
- [ ] Above threshold → size + egress in one line; resumable with `-m`; no IAM/ACL changes, no embedded keys (U5, U4)

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
