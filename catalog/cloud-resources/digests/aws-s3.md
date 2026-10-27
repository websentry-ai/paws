# aws-s3 digest

S3 bucket create, policy, ACL or public-access-block change. Apply before retrying the command.

## Create a bucket: this sequence, one call per step, verify each
1. Create `<org-prefix>-<purpose>-<env>` in `<region>`:
   `aws s3api create-bucket --bucket <name> --region <region> --create-bucket-configuration LocationConstraint=<region>`
2. Block all public access before any object lands:
   `aws s3api put-public-access-block --bucket <name> --public-access-block-configuration BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true`
3. Default encryption (`put-bucket-encryption`): SSE-KMS with the org key when one is named, else SSE-S3 (AES256); say which.
4. Tag with the full set (`put-bucket-tagging`): `owner`=requesting human, `team`=their team, `env`=one of dev, staging, prod, `cost-center`=the one the user gave, else `unassigned` outside prod (say so; ask only when env is prod), `ttl`=today + 90 days (or `permanent`, which needs owner + cost-center to survive review), `created-by=ai-agent`.
5. Versioning `Status=Enabled` for anything holding real data.
6. Read back public-access-block, encryption and tags; report all three in one line; fix gaps.

## Never public by default
- No `put-bucket-acl --acl public-read`, no bucket policy with `"Principal": "*"` Allow, no loosening or deleting the public-access-block without explicit user confirmation this session, restated: "This makes X publicly readable — confirm?"
- Public content goes through CloudFront + Origin Access Control in front of a private bucket. Say so first; only proceed on explicit confirmation after that.

## Policy / ACL changes on an existing bucket
Fetch the current policy, show a before/after diff of principals and actions, then apply. Confirm only if the diff makes anything public.

## Baseline (all cloud resources)
- Least exposure: private, least-privilege by default.
- Tags + TTL go in at creation, not later. Unknown value: use the org default and state it. Cost center defaults to `unassigned`; ask for it only when env is prod.
- Prefer the org's paved road (Terraform module, internal wrapper) over raw CLI; if none is configured, say raw CLI is being used.
- Over $100/month estimated: one-line cost note, then proceed. Resizing or deleting dev resources: state what happens, proceed. Confirm only data-bearing deletes and public exposure.

## Checklist
- [ ] Name follows `<org-prefix>-<purpose>-<env>` (prefix = org short name, lowercased)
- [ ] Public access block + default encryption in the creation sequence
- [ ] Full tag set incl. `cost-center` (`unassigned` outside prod), `ttl` and `created-by=ai-agent`
- [ ] No public ACL / `Principal:*` / PAB loosening without confirmation
- [ ] Post-create read-back reported in one line
