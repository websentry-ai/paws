# gsutil digest

GCS cp / rsync / mv / rm via gsutil or `gcloud storage` (same rules for both). Apply before retrying the command.

Defaults until the org configures them: sanctioned remotes = buckets/remotes already in use on this machine (new ones: ask); external = any bucket not owned by the org (confirm before pushing org data); sensitive = credentials, customer data, financials, unreleased plans.

## 1. Destination is org-owned
- `gsutil ls -L -b gs://<dest-bucket>` (or `gcloud storage buckets describe gs://<dest-bucket>`); check its project is one listed by `gcloud projects list` under the org's domain.
- Bucket outside the sanctioned remotes = cross-boundary: stop and get explicit same-session confirmation before the first byte moves.

## 2. Direction sentence before every cp / rsync / mv
Push or pull, source → destination, inside or outside the org, size. E.g. "Pushing ./build/ → gs://acme-releases/v2.4 (org project acme-prod) — 1.2 GB." `mv` deletes at the source: say so, never call it a copy.

## 3. Dry-run delete-propagating syncs
- `gsutil -m rsync -n <src> <dst>` first; state copy/delete counts; then run the same command without `-n`. No confirmation on top.
- Never `rsync -d` (`gcloud storage rsync --delete-unmatched-destination-objects`) without a same-session dry-run. Pause only if the delete count is wildly out of line with intent.
- `gsutil rm -r` on a prefix: `gsutil ls -r gs://<path>` first, state the object count.

## 4. Classify before moving
`gsutil ls -r gs://<src-path> | head -50`: look for dumps, exports, `.env`/key files, customer-named objects. PII, secrets or customer data never go to a bucket outside the sanctioned remotes. Unclear tier → ask before transferring.

## 5. Size and cost
`gsutil du -s gs://<url>` (`gcloud storage du --summarize`). Above 10 GB: one line with total size, object count, egress implication (GCS → local and cross-cloud are billed; intra-region GCS→GCS is not), then run with `-m`. Uploads are resumable: rerun, don't restart blind.

## Never
- Change IAM/ACLs (`gsutil iam`, `acl`, `signurl`, bucket policy) to fix a transfer error: stop and escalate to the cloud-resources / identity-access skills.
- Embed credentials: no `-o Credentials:gs_service_key_file=...`, no pasted `GOOGLE_APPLICATION_CREDENTIALS` keys, no JSON keys copied into the tree. Auth via `gcloud auth login` / workload identity only.

## Escalation
User wants org data in a personal/external bucket: state the risk in one line, name the compliant path, proceed only on explicit override. External destinations always keep their confirmation; everything else is narrate-and-go.

## Baseline (all bulk transfers)
- U1 direction + endpoints restated; cross-boundary confirmed; moves named as moves.
- U2 delete-propagating syncs dry-run first; stated counts are the gate.
- U3 source spot-checked against sensitivity tiers; unclear → ask first.
- U4 destinations from sanctioned remotes; new remotes narrated (provider, account, scope); credentials only in the tool's config store.
- U5 above threshold: size + egress cost in one line; resumable, rate-limited runs.

## Checklist
- [ ] Destination ownership checked; external confirmed
- [ ] Direction sentence; mv vs cp named
- [ ] `rsync -n` done, counts stated; no `-d` without it
- [ ] Source spot-checked
- [ ] Size + egress stated above 10 GB; no IAM/ACL edits, no embedded keys
